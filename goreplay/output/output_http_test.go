package output

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	_ "net/http/httputil"
	"record-traffic-press/goreplay"
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/plugins"
	"record-traffic-press/goreplay/settings"
	"sync"
	"testing"
)

func TestHTTPOutput(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("User-Agent") != "Gor" {
			t.Error("Wrong header")
		}

		if req.Method == "OPTIONS" {
			t.Error("Wrong method")
		}

		if req.Method == "POST" {
			defer req.Body.Close()
			body, _ := ioutil.ReadAll(req.Body)

			if string(body) != "a=1&b=2" {
				t.Error("Wrong POST body:", string(body))
			}
		}

		wg.Done()
	}))
	defer server.Close()

	headers := settings.HTTPHeaders{goreplay.httpHeader{"User-Agent", "Gor"}}
	methods := settings.HTTPMethods{[]byte("GET"), []byte("PUT"), []byte("POST")}
	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{Headers: headers, Methods: methods}

	httpOutput := NewHTTPOutput(server.URL, &HTTPOutputConfig{TrackResponses: false})
	output := bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{httpOutput, output},
	}
	plugins.All = append(plugins.All, input, output, httpOutput)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 10; i++ {
		// 2 http-output, 2 - test output request
		wg.Add(4) // OPTIONS should be ignored
		input.EmitPOST()
		input.EmitOPTIONS()
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()

	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{}
}

func TestHTTPOutputKeepOriginalHost(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != "custom-host.com" {
			t.Error("Wrong header", req.Host)
		}

		wg.Done()
	}))
	defer server.Close()

	headers := settings.HTTPHeaders{goreplay.httpHeader{"Host", "custom-host.com"}}
	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{Headers: headers}

	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{OriginalHost: true, SkipVerify: true})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	wg.Add(1)
	input.EmitGET()

	wg.Wait()
	emitter.Close()
	settings.Settings.ModifierConfig = settings.HTTPModifierConfig{}
}

func TestHTTPOutputSSL(t *testing.T) {
	wg := new(sync.WaitGroup)

	// Origing and Replay server initialization
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))

	input := bootstrap.NewTestInput()
	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{SkipVerify: true})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	wg.Add(2)

	input.EmitPOST()
	input.EmitGET()

	wg.Wait()
	emitter.Close()
}

func TestHTTPOutputSessions(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()
	input.skipHeader = true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		wg.Done()
	}))
	defer server.Close()

	settings.Settings.RecognizeTCPSessions = true
	settings.Settings.SplitOutput = true
	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)
	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	uuid1 := []byte("1234567890123456789a0000")
	uuid2 := []byte("1234567890123456789d0000")

	for i := 0; i < 10; i++ {
		wg.Add(1) // OPTIONS should be ignored
		copy(uuid1[20:], goreplay.randByte(4))
		input.EmitBytes([]byte("1 " + string(uuid1) + " 1\n" + "GET / HTTP/1.1\r\n\r\n"))
	}

	for i := 0; i < 10; i++ {
		wg.Add(1) // OPTIONS should be ignored
		copy(uuid2[20:], goreplay.randByte(4))
		input.EmitBytes([]byte("1 " + string(uuid2) + " 1\n" + "GET / HTTP/1.1\r\n\r\n"))
	}

	wg.Wait()

	emitter.Close()

	settings.Settings.RecognizeTCPSessions = false
	settings.Settings.SplitOutput = false
}

func BenchmarkHTTPOutput(b *testing.B) {
	wg := new(sync.WaitGroup)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))
	defer server.Close()

	input := bootstrap.NewTestInput()
	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{WorkersMax: 1})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		input.EmitPOST()
	}

	wg.Wait()
	emitter.Close()
}

func BenchmarkHTTPOutputTLS(b *testing.B) {
	wg := new(sync.WaitGroup)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))
	defer server.Close()

	input := bootstrap.NewTestInput()
	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{SkipVerify: true, WorkersMax: 1})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		input.EmitPOST()
	}

	wg.Wait()
	emitter.Close()
}
