package output

import (
	"log"
	"net/http"
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/settings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketOutput(t *testing.T) {
	wg := new(sync.WaitGroup)

	var gotHeader http.Header
	wsAddr := startWebsocket(func(data []byte) {
		wg.Done()
	}, func(header http.Header) {
		gotHeader = header
	})
	input := bootstrap.NewTestInput()
	headers := map[string][]string{
		"key1": {"value1"},
		"key2": {"value2"},
	}
	output := NewWebSocketOutput(wsAddr, &WebSocketOutputConfig{Workers: 1, Headers: headers})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()

	if assert.NotNil(t, gotHeader) {
		assert.Equal(t, "Basic dXNlcjE=", gotHeader.Get("Authorization"))
		for k, values := range headers {
			assert.Equal(t, 1, len(values))
			assert.Equal(t, values[0], gotHeader.Get(k))
		}
	}
}

func startWebsocket(cb func([]byte), headercb func(http.Header)) string {
	upgrader := websocket.Upgrader{}

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		headercb(r.Header)
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		go func(conn *websocket.Conn) {
			defer conn.Close()
			for {
				_, msg, _ := conn.ReadMessage()
				cb(msg)
			}
		}(c)
	})

	go func() {
		err := http.ListenAndServe("localhost:8081", nil)
		if err != nil {
			log.Fatal("Can't start:", err)
		}
	}()

	return "ws://user1@localhost:8081/test"
}
