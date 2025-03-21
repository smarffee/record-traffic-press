package core

import (
	input2 "record-traffic-press/goreplay/input"
	output2 "record-traffic-press/goreplay/output"
	"record-traffic-press/goreplay/settings"
	"testing"
)

func TestPluginsRegistration(t *testing.T) {
	settings.Settings.InputDummy = []string{"[]"}
	settings.Settings.OutputDummy = []string{"[]"}
	settings.Settings.OutputHTTP = []string{"www.example.com|10"}
	settings.Settings.InputFile = []string{"/dev/null"}

	plugins := NewPlugins()

	if len(plugins.Inputs) != 3 {
		t.Errorf("Should be 3 inputs got %d", len(plugins.Inputs))
	}

	if _, ok := plugins.Inputs[0].(*input2.DummyInput); !ok {
		t.Errorf("First input should be DummyInput")
	}

	if _, ok := plugins.Inputs[1].(*input2.FileInput); !ok {
		t.Errorf("Second input should be FileInput")
	}

	if len(plugins.Outputs) != 2 {
		t.Errorf("Should be 2 output %d", len(plugins.Outputs))
	}

	if _, ok := plugins.Outputs[0].(*output2.DummyOutput); !ok {
		t.Errorf("First output should be DummyOutput")
	}

	if l, ok := plugins.Outputs[1].(*Limiter); ok {
		if _, ok := l.plugin.(*output2.HTTPOutput); !ok {
			t.Errorf("HTTPOutput should be wrapped in limiter")
		}
	} else {
		t.Errorf("Second output should be Limiter")
	}

}
