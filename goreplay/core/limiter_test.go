//go:build !race

package core

import (
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/plugins"
	"record-traffic-press/goreplay/settings"
	"sync"
	"testing"
)

func TestOutputLimiter(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()
	output := NewLimiter(bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	}), "10")
	wg.Add(10)

	plugins := &InOutPlugins{
		Inputs:  []PluginReader{input},
		Outputs: []PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 100; i++ {
		input.EmitGET()
	}

	wg.Wait()
	emitter.Close()
}

func TestInputLimiter(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := NewLimiter(bootstrap.NewTestInput(), "10")
	output := bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	})
	wg.Add(10)

	plugins := &InOutPlugins{
		Inputs:  []PluginReader{input},
		Outputs: []PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 100; i++ {
		input.(*Limiter).plugin.(*plugins.TestInput).EmitGET()
	}

	wg.Wait()
	emitter.Close()
}

// Should limit all requests
func TestPercentLimiter1(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()
	output := NewLimiter(bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	}), "0%")

	plugins := &InOutPlugins{
		Inputs:  []PluginReader{input},
		Outputs: []PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 100; i++ {
		input.EmitGET()
	}

	wg.Wait()
}

// Should not limit at all
func TestPercentLimiter2(t *testing.T) {
	wg := new(sync.WaitGroup)

	input := bootstrap.NewTestInput()
	output := NewLimiter(bootstrap.NewTestOutput(func(*plugins.Message) {
		wg.Done()
	}), "100%")
	wg.Add(100)

	plugins := &InOutPlugins{
		Inputs:  []PluginReader{input},
		Outputs: []PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	for i := 0; i < 100; i++ {
		input.EmitGET()
	}

	wg.Wait()
}
