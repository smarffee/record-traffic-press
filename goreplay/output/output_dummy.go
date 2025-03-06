package output

import (
	"os"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/input"
)

// DummyOutput used for debugging, prints all incoming requests
type DummyOutput struct {
}

// NewDummyOutput constructor for DummyOutput
func NewDummyOutput() (di *DummyOutput) {
	di = new(DummyOutput)

	return
}

// PluginWrite writes message to this plugin
func (i *DummyOutput) PluginWrite(msg *common.Message) (int, error) {
	var n, nn int
	var err error
	n, err = os.Stdout.Write(msg.Meta)
	nn, err = os.Stdout.Write(msg.Data)
	n += nn
	nn, err = os.Stdout.Write(input.PayloadSeparatorAsBytes)
	n += nn

	return n, err
}

func (i *DummyOutput) String() string {
	return "Dummy Output"
}
