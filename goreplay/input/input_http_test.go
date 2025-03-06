package input

import (
	"bytes"
	"net/http"
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/plugins"
	"record-traffic-press/goreplay/settings"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHTTPInput(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewHTTPInput("127.0.0.1:0")
	time.Sleep(time.Millisecond)
	output := bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	address := strings.Replace(input.address, "[::]", "127.0.0.1", -1)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		http.Get("http://" + address + "/")
	}

	wg.Wait()
	emitter.Close()
}

func TestInputHTTPLargePayload(t *testing.T) {
	wg := new(sync.WaitGroup)
	const n = 10 << 20 // 10MB
	var large [n]byte
	large[n-1] = '0'

	input := NewHTTPInput("127.0.0.1:0")
	output := bootstrap.NewTestOutput(func(msg *plugins.Message) {
		_len := len(msg.Data)
		if _len >= n { // considering http body CRLF
			t.Errorf("expected body to be >= %d", n)
		}
		wg.Done()
	})
	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	defer emitter.Close()
	go emitter.Start(plugins, settings.Settings.Middleware)

	address := strings.Replace(input.address, "[::]", "127.0.0.1", -1)
	var req *http.Request
	var err error
	req, err = http.NewRequest("POST", "http://"+address, bytes.NewBuffer(large[:]))
	if err != nil {
		t.Error(err)
		return
	}
	wg.Add(1)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	wg.Wait()
}
