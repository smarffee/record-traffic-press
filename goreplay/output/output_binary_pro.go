package output

import (
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/glogs"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"sync/atomic"
	"time"
)

// 是一种编译时检查，确保 BinaryOutput 类型实现了 bootstrap.PluginWriter 接口
var _ core.PluginWriter = (*BinaryOutput)(nil)

// BinaryOutput plugin manage pool of workers which send request to replayed server
// By default workers pool is dynamic and starts with 10 workers
// You can specify fixed number of workers using `--output-tcp-workers`
type BinaryOutput struct {
	// Keep this as first element of struct because it guarantees 64bit
	// alignment. atomic.* functions crash on 32bit machines if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	activeWorkers int64
	address       string
	queue         chan *common.Message
	responses     chan response
	needWorker    chan int
	quit          chan struct{}
	config        *settings.BinaryOutputConfig
	queueStats    *core.GorStat
}

// NewBinaryOutput constructor for BinaryOutput
// Initialize workers
func NewBinaryOutput(address string, config *settings.BinaryOutputConfig) core.PluginReadWriter {
	o := new(BinaryOutput)

	o.address = address
	o.config = config

	o.queue = make(chan *common.Message, 1000)
	o.responses = make(chan response, 1000)
	o.needWorker = make(chan int, 1)
	o.quit = make(chan struct{})

	// Initial workers count
	if o.config.Workers == 0 {
		o.needWorker <- common.InitialDynamicWorkers
	} else {
		o.needWorker <- o.config.Workers
	}

	go o.workerMaster()

	return o
}

func (o *BinaryOutput) workerMaster() {
	for {
		newWorkers := <-o.needWorker
		for i := 0; i < newWorkers; i++ {
			go o.startWorker()
		}

		// Disable dynamic scaling if workers poll fixed size
		if o.config.Workers != 0 {
			return
		}
	}
}

func (o *BinaryOutput) startWorker() {
	client := bootstrap.NewTCPClient(o.address, &bootstrap.TCPClientConfig{
		Debug:              o.config.Debug,
		Timeout:            o.config.Timeout,
		ResponseBufferSize: int(o.config.BufferSize),
	})

	deathCount := 0

	atomic.AddInt64(&o.activeWorkers, 1)

	for {
		select {
		case msg := <-o.queue:
			o.sendRequest(client, msg)
			deathCount = 0
		case <-time.After(time.Millisecond * 100):
			// When dynamic scaling enabled workers die after 2s of inactivity
			if o.config.Workers == 0 {
				deathCount++
			} else {
				continue
			}

			if deathCount > 20 {
				workersCount := atomic.LoadInt64(&o.activeWorkers)

				// At least 1 startWorker should be alive
				if workersCount != 1 {
					atomic.AddInt64(&o.activeWorkers, -1)
					return
				}
			}
		}
	}
}

// PluginWrite writes a message tothis plugin
func (o *BinaryOutput) PluginWrite(msg *common.Message) (n int, err error) {
	if !proto.IsRequestPayload(msg.Meta) {
		return len(msg.Data), nil
	}

	o.queue <- msg

	if o.config.Workers == 0 {
		workersCount := atomic.LoadInt64(&o.activeWorkers)

		if len(o.queue) > int(workersCount) {
			o.needWorker <- len(o.queue)
		}
	}

	return len(msg.Data) + len(msg.Meta), nil
}

// PluginRead reads a message from this plugin
func (o *BinaryOutput) PluginRead() (*common.Message, error) {
	var resp response
	var msg common.Message
	select {
	case <-o.quit:
		return nil, common.ErrorStopped
	case resp = <-o.responses:
	}
	msg.Data = resp.payload
	msg.Meta = proto.PayloadHeader(proto.ReplayedResponsePayload, resp.uuid, resp.startedAt, resp.roundTripTime)

	return &msg, nil
}

func (o *BinaryOutput) sendRequest(client *bootstrap.TCPClient, msg *common.Message) {
	if !proto.IsRequestPayload(msg.Meta) {
		return
	}

	uuid := proto.PayloadID(msg.Meta)

	start := time.Now()
	resp, err := client.Send(msg.Data)
	stop := time.Now()

	if err != nil {
		glogs.Debug(1, "Request error:", err)
	}

	if o.config.TrackResponses {
		o.responses <- response{resp, uuid, start.UnixNano(), stop.UnixNano() - start.UnixNano()}
	}
}

func (o *BinaryOutput) String() string {
	return "Binary output: " + o.address
}

// Close closes this plugin for reading
func (o *BinaryOutput) Close() error {
	close(o.quit)
	return nil
}
