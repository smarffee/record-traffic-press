package bootstrap

import (
	"fmt"
	"io"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/glogs"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"record-traffic-press/goreplay/utils"
	"sync"

	"github.com/coocood/freecache"
)

// Emitter represents an abject to manage plugins communication
type Emitter struct {
	sync.WaitGroup
	plugins *core.InOutPlugins
}

// NewEmitter creates and initializes new Emitter object.
func NewEmitter() *Emitter {
	return &Emitter{}
}

// Start initialize loop for sending data from inputs to outputs
func (e *Emitter) Start(plugins *core.InOutPlugins, middlewareCmd string) {
	if settings.Settings.CopyBufferSize < 1 {
		settings.Settings.CopyBufferSize = 5 << 20
	}
	e.plugins = plugins

	if middlewareCmd != "" {
		middleware := core.NewMiddleware(middlewareCmd)

		for _, in := range plugins.Inputs {
			middleware.ReadFrom(in)
		}

		e.plugins.Inputs = append(e.plugins.Inputs, middleware)
		e.plugins.All = append(e.plugins.All, middleware)
		e.Add(1)
		go func() {
			defer e.Done()
			if err := CopyMulty(middleware, plugins.Outputs...); err != nil {
				glogs.Debug(2, fmt.Sprintf("[EMITTER] error during copy: %q", err))
			}
		}()
	} else {
		for _, in := range plugins.Inputs {
			e.Add(1)
			go func(in core.PluginReader) {
				defer e.Done()
				if err := CopyMulty(in, plugins.Outputs...); err != nil {
					glogs.Debug(2, fmt.Sprintf("[EMITTER] error during copy: %q", err))
				}
			}(in)
		}
	}
}

// Close closes all the goroutine and waits for it to finish.
func (e *Emitter) Close() {
	for _, p := range e.plugins.All {
		if cp, ok := p.(io.Closer); ok {
			cp.Close()
		}
	}
	if len(e.plugins.All) > 0 {
		// wait for everything to stop
		e.Wait()
	}
	e.plugins.All = nil // avoid Close to make changes again
}

// CopyMulty copies from 1 reader to multiple writers
func CopyMulty(src core.PluginReader, writers ...core.PluginWriter) error {

	modifier := core.NewHTTPModifier(&settings.Settings.ModifierConfig)
	filteredRequests := freecache.NewCache(200 * 1024 * 1024) // 200M

	for {
		msg, err := src.PluginRead()
		if err != nil {
			if err == common.ErrorStopped || err == io.EOF {
				return nil
			}
			return err
		}
		if msg != nil && len(msg.Data) > 0 {
			if len(msg.Data) > int(settings.Settings.CopyBufferSize) {
				msg.Data = msg.Data[:settings.Settings.CopyBufferSize]
			}
			meta := proto.PayloadMeta(msg.Meta)
			if len(meta) < 3 {
				glogs.Debug(2, fmt.Sprintf("[EMITTER] Found malformed record %q from %q", msg.Meta, src))
				continue
			}
			requestID := meta[1]
			// start a subroutine only when necessary
			if settings.Settings.Verbose >= 3 {
				glogs.Debug(3, "[EMITTER] input: ", utils.SliceToString(msg.Meta[:len(msg.Meta)-1]), " from: ", src)
			}
			if modifier != nil {
				glogs.Debug(3, "[EMITTER] modifier:", requestID, "from:", src)
				if proto.IsRequestPayload(msg.Meta) {
					msg.Data = modifier.Rewrite(msg.Data)
					// If modifier tells to skip request
					if len(msg.Data) == 0 {
						filteredRequests.Set(requestID, []byte{}, 60) //
						continue
					}
					glogs.Debug(3, "[EMITTER] Rewritten input:", requestID, "from:", src)

				} else {
					_, err := filteredRequests.Get(requestID)
					if err == nil {
						filteredRequests.Del(requestID)
						continue
					}
				}
			}

			if settings.Settings.PrettifyHTTP {
				msg.Data = core.PrettifyHTTP(msg.Data)
				if len(msg.Data) == 0 {
					continue
				}
			}

			for _, dst := range writers {
				if _, err := dst.PluginWrite(msg); err != nil && err != io.ErrClosedPipe {
					return err
				}
			}
		}
	}
}
