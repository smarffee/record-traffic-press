package output

import (
	"record-traffic-press/goreplay/common"
)

// NullOutput used for debugging, prints nothing
type NullOutput struct {
}

// NewNullOutput constructor for NullOutput
func NewNullOutput() (o *NullOutput) {
	return new(NullOutput)
}

// PluginWrite writes message to this plugin
func (o *NullOutput) PluginWrite(msg *common.Message) (int, error) {
	return len(msg.Data) + len(msg.Meta), nil
}

func (o *NullOutput) String() string {
	return "Null Output"
}
