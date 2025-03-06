package bootstrap

import (
	"fmt"
	"os"
	"record-traffic-press/goreplay"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/plugins"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	goreplay.PRO = true
	code := m.Run()
	os.Exit(code)
}

func TestEmitter(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewTestInput()
	output := NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()
}

func TestEmitterFiltered(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewTestInput()
	input.skipHeader = true

	output := NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	methods := settings.HTTPMethods{[]byte("GET")}
	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{Methods: methods}

	emitter := &Emitter{}
	go emitter.Start(plugins, "")

	wg.Add(2)

	id := proto.uuid()
	reqh := proto.payloadHeader(proto.RequestPayload, id, time.Now().UnixNano(), -1)
	reqb := append(reqh, []byte("POST / HTTP/1.1\r\nHost: www.w3.org\r\nUser-Agent: Go 1.1 package http\r\nAccept-Encoding: gzip\r\n\r\n")...)

	resh := proto.payloadHeader(proto.ResponsePayload, id, time.Now().UnixNano()+1, 1)
	respb := append(resh, []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")...)

	input.EmitBytes(reqb)
	input.EmitBytes(respb)

	id = proto.uuid()
	reqh = proto.payloadHeader(proto.RequestPayload, id, time.Now().UnixNano(), -1)
	reqb = append(reqh, []byte("GET / HTTP/1.1\r\nHost: www.w3.org\r\nUser-Agent: Go 1.1 package http\r\nAccept-Encoding: gzip\r\n\r\n")...)

	resh = proto.payloadHeader(proto.ResponsePayload, id, time.Now().UnixNano()+1, 1)
	respb = append(resh, []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")...)

	input.EmitBytes(reqb)
	input.EmitBytes(respb)

	wg.Wait()
	emitter.Close()

	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{}
}

func TestEmitterSplitRoundRobin(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewTestInput()

	var counter1, counter2 int32

	output1 := NewTestOutput(func(*plugins.Message) {
		atomic.AddInt32(&counter1, 1)
		wg.Done()
	})

	output2 := NewTestOutput(func(*plugins.Message) {
		atomic.AddInt32(&counter2, 1)
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output1, output2},
	}

	settings.Settings.SplitOutput = true

	emitter := NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()

	emitter.Close()

	if counter1 == 0 || counter2 == 0 || counter1 != counter2 {
		t.Errorf("Round robin should split traffic equally: %d vs %d", counter1, counter2)
	}

	settings.Settings.SplitOutput = false
}

func TestEmitterRoundRobin(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewTestInput()

	var counter1, counter2 int32

	output1 := NewTestOutput(func(*plugins.Message) {
		counter1++
		wg.Done()
	})

	output2 := NewTestOutput(func(*plugins.Message) {
		counter2++
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output1, output2},
	}
	plugins.All = append(plugins.All, input, output1, output2)

	settings.Settings.SplitOutput = true

	emitter := NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()

	if counter1 == 0 || counter2 == 0 {
		t.Errorf("Round robin should split traffic equally: %d vs %d", counter1, counter2)
	}

	settings.Settings.SplitOutput = false
}

func TestEmitterSplitSession(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(200)

	input := NewTestInput()
	input.skipHeader = true

	var counter1, counter2 int32

	output1 := NewTestOutput(func(msg *plugins.Message) {
		if proto.payloadID(msg.Meta)[0] == 'a' {
			counter1++
		}
		wg.Done()
	})

	output2 := NewTestOutput(func(msg *plugins.Message) {
		if proto.payloadID(msg.Meta)[0] == 'b' {
			counter2++
		}
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output1, output2},
	}

	settings.Settings.SplitOutput = true
	settings.Settings.RecognizeTCPSessions = true

	emitter := NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 200; i++ {
		// Keep session but randomize
		id := make([]byte, 20)
		if i&1 == 0 { // for recognizeTCPSessions one should be odd and other will be even number
			id[0] = 'a'
		} else {
			id[0] = 'b'
		}
		input.EmitBytes([]byte(fmt.Sprintf("1 %s 1 1\nGET / HTTP/1.1\r\n\r\n", id[:20])))
	}

	wg.Wait()

	if counter1 != counter2 {
		t.Errorf("Round robin should split traffic equally: %d vs %d", counter1, counter2)
	}

	settings.Settings.SplitOutput = false
	settings.Settings.RecognizeTCPSessions = false
	emitter.Close()
}

func BenchmarkEmitter(b *testing.B) {
	wg := new(sync.WaitGroup)

	input := NewTestInput()

	output := NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()
}
