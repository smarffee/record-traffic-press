package output

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/glogs"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"sync/atomic"
	"time"
)

type response struct {
	payload       []byte
	uuid          []byte
	startedAt     int64
	roundTripTime int64
}

// HTTPOutput plugin manage pool of workers which send request to replayed server
// By default workers pool is dynamic and starts with 1 worker or workerMin workers
// You can specify maximum number of workers using `--output-http-workers`
type HTTPOutput struct {
	activeWorkers  int64
	config         *settings.HTTPOutputConfig
	queueStats     *core.GorStat
	elasticSearch  *common.ESPlugin
	client         *HTTPClient
	stopWorker     chan struct{}
	queue          chan *common.Message
	responses      chan *response
	stop           chan bool // Channel used only to indicate goroutine should shutdown
	workerSessions map[string]*httpWorker
}

type httpWorker struct {
	output       *HTTPOutput
	client       *HTTPClient
	lastActivity time.Time
	queue        chan *common.Message
	stop         chan bool
}

func newHTTPWorker(output *HTTPOutput, queue chan *common.Message) *httpWorker {
	client := NewHTTPClient(output.config)

	w := &httpWorker{client: client, output: output}
	if queue == nil {
		w.queue = make(chan *common.Message, 100)
	} else {
		w.queue = queue
	}
	w.stop = make(chan bool)

	go func() {
		for {
			select {
			case msg := <-w.queue:
				output.sendRequest(client, msg)
			case <-w.stop:
				return
			}
		}
	}()

	return w
}

// NewHTTPOutput constructor for HTTPOutput
// Initialize workers
func NewHTTPOutput(address string, config *settings.HTTPOutputConfig) core.PluginReadWriter {
	o := new(HTTPOutput)
	var err error
	newConfig := config.Copy()
	newConfig.Url, err = url.Parse(address)
	if err != nil {
		log.Fatal(fmt.Sprintf("[OUTPUT-HTTP] parse HTTP output URL error[%q]", err))
	}
	if newConfig.Url.Scheme == "" {
		newConfig.Url.Scheme = "http"
	}
	newConfig.RawURL = newConfig.Url.String()
	if newConfig.Timeout < time.Millisecond*100 {
		newConfig.Timeout = time.Second
	}
	if newConfig.BufferSize <= 0 {
		newConfig.BufferSize = 100 * 1024 // 100kb
	}
	if newConfig.WorkersMin <= 0 {
		newConfig.WorkersMin = 1
	}
	if newConfig.WorkersMin > 1000 {
		newConfig.WorkersMin = 1000
	}
	if newConfig.WorkersMax <= 0 {
		newConfig.WorkersMax = math.MaxInt32 // ideally so large
	}
	if newConfig.WorkersMax < newConfig.WorkersMin {
		newConfig.WorkersMax = newConfig.WorkersMin
	}
	if newConfig.QueueLen <= 0 {
		newConfig.QueueLen = 1000
	}
	if newConfig.RedirectLimit < 0 {
		newConfig.RedirectLimit = 0
	}
	if newConfig.WorkerTimeout <= 0 {
		newConfig.WorkerTimeout = time.Second * 2
	}
	o.config = newConfig
	o.stop = make(chan bool)
	if o.config.Stats {
		o.queueStats = core.NewGorStat("output_http", o.config.StatsMs)
	}

	o.queue = make(chan *common.Message, o.config.QueueLen)
	if o.config.TrackResponses {
		o.responses = make(chan *response, o.config.QueueLen)
	}
	// it should not be buffered to avoid races
	o.stopWorker = make(chan struct{})

	if o.config.ElasticSearch != "" {
		o.elasticSearch = new(common.ESPlugin)
		o.elasticSearch.Init(o.config.ElasticSearch)
	}
	o.client = NewHTTPClient(o.config)

	o.activeWorkers += int64(o.config.WorkersMin)
	for i := 0; i < o.config.WorkersMin; i++ {
		go o.startWorker()
	}
	go o.workerMaster()

	return o
}

func (o *HTTPOutput) workerMaster() {
	var timer = time.NewTimer(o.config.WorkerTimeout)
	defer func() {
		// recover from panics caused by trying to send in
		// a closed chan(o.stopWorker)
		recover()
	}()
	defer timer.Stop()
	for {
		select {
		case <-o.stop:
			return
		default:
			<-timer.C
		}
		// rollback workers
	rollback:
		if atomic.LoadInt64(&o.activeWorkers) > int64(o.config.WorkersMin) && len(o.queue) < 1 {
			// close one worker
			o.stopWorker <- struct{}{}
			atomic.AddInt64(&o.activeWorkers, -1)
			goto rollback
		}
		timer.Reset(o.config.WorkerTimeout)
	}
}

func (o *HTTPOutput) sessionWorkerMaster() {
	gc := time.Tick(time.Second)

	for {
		select {
		case msg := <-o.queue:
			id := proto.PayloadID(msg.Meta)
			sessionID := string(id[0:20])
			worker, ok := o.workerSessions[sessionID]

			if !ok {
				atomic.AddInt64(&o.activeWorkers, 1)
				worker = newHTTPWorker(o, nil)
				o.workerSessions[sessionID] = worker
			}

			worker.queue <- msg
			worker.lastActivity = time.Now()
		case <-gc:
			now := time.Now()

			for id, w := range o.workerSessions {
				if !w.lastActivity.IsZero() && now.Sub(w.lastActivity) >= 120*time.Second {
					w.stop <- true
					delete(o.workerSessions, id)
					atomic.AddInt64(&o.activeWorkers, -1)
				}
			}
		}
	}
}

func (o *HTTPOutput) startWorker() {
	for {
		select {
		case <-o.stopWorker:
			return
		case msg := <-o.queue:
			o.sendRequest(o.client, msg)
		}
	}
}

// PluginWrite writes message to this plugin
func (o *HTTPOutput) PluginWrite(msg *common.Message) (n int, err error) {
	if !proto.IsRequestPayload(msg.Meta) {
		return len(msg.Data), nil
	}

	select {
	case <-o.stop:
		return 0, common.ErrorStopped
	case o.queue <- msg:
	}

	if o.config.Stats {
		o.queueStats.Write(len(o.queue))
	}

	if o.config.WorkersMax != o.config.WorkersMin {
		workersCount := int(atomic.LoadInt64(&o.activeWorkers))

		if len(o.queue) > workersCount {
			extraWorkersReq := len(o.queue) - workersCount + 1
			maxWorkersAvailable := o.config.WorkersMax - workersCount
			if extraWorkersReq > maxWorkersAvailable {
				extraWorkersReq = maxWorkersAvailable
			}
			if extraWorkersReq > 0 {
				for i := 0; i < extraWorkersReq; i++ {
					go o.startWorker()
					atomic.AddInt64(&o.activeWorkers, 1)
				}
			}
		}
	}

	return len(msg.Data) + len(msg.Meta), nil
}

// PluginRead reads message from this plugin
func (o *HTTPOutput) PluginRead() (*common.Message, error) {
	if !o.config.TrackResponses {
		return nil, common.ErrorStopped
	}
	var resp *response
	var msg common.Message
	select {
	case <-o.stop:
		return nil, common.ErrorStopped
	case resp = <-o.responses:
		msg.Data = resp.payload
	}

	msg.Meta = proto.PayloadHeader(proto.ReplayedResponsePayload, resp.uuid, resp.startedAt, resp.roundTripTime)

	return &msg, nil
}

func (o *HTTPOutput) sendRequest(client *HTTPClient, msg *common.Message) {
	if !proto.IsRequestPayload(msg.Meta) {
		return
	}

	uuid := proto.PayloadID(msg.Meta)
	start := time.Now()
	resp, err := client.Send(msg.Data)
	stop := time.Now()

	if err != nil {
		glogs.Debug(1, fmt.Sprintf("[HTTP-OUTPUT] error when sending: %q", err))
		return
	}
	if resp == nil {
		return
	}

	if o.config.TrackResponses {
		o.responses <- &response{resp, uuid, start.UnixNano(), stop.UnixNano() - start.UnixNano()}
	}

	if o.elasticSearch != nil {
		o.elasticSearch.ResponseAnalyze(msg.Data, resp, start, stop)
	}
}

func (o *HTTPOutput) String() string {
	return "HTTP output: " + o.config.RawURL
}

// Close closes the data channel so that data
func (o *HTTPOutput) Close() error {
	close(o.stop)
	close(o.stopWorker)
	return nil
}

// HTTPClient holds configurations for a single HTTP client
type HTTPClient struct {
	config *settings.HTTPOutputConfig
	Client *http.Client
}

// NewHTTPClient returns new http client with check redirects policy
func NewHTTPClient(config *settings.HTTPOutputConfig) *HTTPClient {
	client := new(HTTPClient)
	client.config = config
	var transport *http.Transport
	client.Client = &http.Client{
		Timeout: client.config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= client.config.RedirectLimit {
				glogs.Debug(1, fmt.Sprintf("[HTTPCLIENT] maximum output-http-redirects[%d] reached!", client.config.RedirectLimit))
				return http.ErrUseLastResponse
			}
			lastReq := via[len(via)-1]
			resp := req.Response
			glogs.Debug(2, fmt.Sprintf("[HTTPCLIENT] HTTP redirects from %q to %q with %q", lastReq.Host, req.Host, resp.Status))
			return nil
		},
	}
	if config.SkipVerify {
		// clone to avoid modifying global default RoundTripper
		transport = http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Client.Transport = transport
	}

	return client
}

// Send sends an http request using client created by NewHTTPClient
func (c *HTTPClient) Send(data []byte) ([]byte, error) {
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		return nil, err
	}
	// we don't send CONNECT or OPTIONS request
	if req.Method == http.MethodConnect {
		return nil, nil
	}

	if !c.config.OriginalHost {
		req.Host = c.config.Url.Host
	}

	// fix #862
	if c.config.Url.Path == "" && c.config.Url.RawQuery == "" {
		req.URL.Scheme = c.config.Url.Scheme
		req.URL.Host = c.config.Url.Host
	} else {
		req.URL = c.config.Url
	}

	// force connection to not be closed, which can affect the global client
	req.Close = false
	// it's an error if this is not equal to empty string
	req.RequestURI = ""

	resp, err = c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if c.config.TrackResponses {
		return httputil.DumpResponse(resp, true)
	}
	_ = resp.Body.Close()
	return nil, nil
}
