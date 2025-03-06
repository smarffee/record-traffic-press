package bootstrap

import (
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core"
)

type writeCallback func(*common.Message)

// TestOutput used in testing to intercept any output into callback
type TestOutput struct {
	cb writeCallback
}

// NewTestOutput constructor for TestOutput, accepts callback which get called on each incoming Write
func NewTestOutput(cb writeCallback) core.PluginWriter {
	i := new(TestOutput)
	i.cb = cb

	return i
}

// PluginWrite write message to this plugin
func (i *TestOutput) PluginWrite(msg *common.Message) (int, error) {
	i.cb(msg)

	return len(msg.Data) + len(msg.Meta), nil
}

func (i *TestOutput) String() string {
	return "Test Output"
}
