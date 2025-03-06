package output

import (
	"context"
	"crypto/tls"
	"fmt"
	"hash/fnv"
	"net"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/glogs"
	"record-traffic-press/goreplay/input"
	"record-traffic-press/goreplay/proto"
	"record-traffic-press/goreplay/settings"
	"time"
)

// TCPOutput used for sending raw tcp payloads
// Currently used for core communication between listener and replay server
// Can be used for transferring binary payloads like protocol buffers
type TCPOutput struct {
	address     string
	limit       int
	buf         []chan *common.Message
	bufStats    *core.GorStat
	config      *settings.TCPOutputConfig
	workerIndex uint32

	close bool
}

// NewTCPOutput constructor for TCPOutput
// Initialize X workers which hold keep-alive connection
func NewTCPOutput(address string, config *settings.TCPOutputConfig) core.PluginWriter {
	o := new(TCPOutput)

	o.address = address
	o.config = config

	if settings.Settings.OutputTCPStats {
		o.bufStats = core.NewGorStat("output_tcp", 5000)
	}

	// create X buffers and send the buffer index to the worker
	o.buf = make([]chan *common.Message, o.config.Workers)
	for i := 0; i < o.config.Workers; i++ {
		o.buf[i] = make(chan *common.Message, 100)
		go o.worker(i)
	}

	return o
}

func (o *TCPOutput) worker(bufferIndex int) {
	retries := 0
	conn, err := o.connect(o.address)
	for {
		if o.close {
			return
		}

		if err == nil {
			break
		}

		glogs.Debug(1, fmt.Sprintf("Can't connect to aggregator instance, reconnecting in 1 second. Retries:%d", retries))
		time.Sleep(1 * time.Second)

		conn, err = o.connect(o.address)
		retries++
	}

	if retries > 0 {
		glogs.Debug(2, fmt.Sprintf("Connected to aggregator instance after %d retries", retries))
	}

	defer conn.Close()

	if o.config.GetInitMessage != nil {
		msg := o.config.GetInitMessage()
		_ = o.writeToConnection(conn, msg)
	}

	for {
		msg := <-o.buf[bufferIndex]
		err = o.writeToConnection(conn, msg)
		if err != nil {
			glogs.Debug(2, "INFO: TCP output connection closed, reconnecting")
			go o.worker(bufferIndex)
			o.buf[bufferIndex] <- msg
			break
		}
	}
}

func (o *TCPOutput) writeToConnection(conn net.Conn, msg *common.Message) (err error) {
	if o.config.WriteBeforeMessage != nil {
		err = o.config.WriteBeforeMessage(conn, msg)
	}

	if err == nil {
		if _, err = conn.Write(msg.Meta); err == nil {
			if _, err = conn.Write(msg.Data); err == nil {
				_, err = conn.Write(input.PayloadSeparatorAsBytes)
			}
		}
	}

	return err
}

func (o *TCPOutput) getBufferIndex(msg *common.Message) int {
	if !o.config.Sticky {
		o.workerIndex++
		return int(o.workerIndex) % o.config.Workers
	}

	hasher := fnv.New32a()
	hasher.Write(proto.PayloadID(msg.Meta))
	return int(hasher.Sum32()) % o.config.Workers
}

// PluginWrite writes message to this plugin
func (o *TCPOutput) PluginWrite(msg *common.Message) (n int, err error) {
	if !proto.IsOriginPayload(msg.Meta) {
		return len(msg.Data), nil
	}

	bufferIndex := o.getBufferIndex(msg)
	o.buf[bufferIndex] <- msg

	if settings.Settings.OutputTCPStats {
		o.bufStats.Write(len(o.buf[bufferIndex]))
	}

	return len(msg.Data) + len(msg.Meta), nil
}

func (o *TCPOutput) connect(address string) (conn net.Conn, err error) {
	if o.config.Secure {
		var d tls.Dialer
		d.Config = &tls.Config{InsecureSkipVerify: o.config.SkipVerify}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err = d.DialContext(ctx, "tcp", address)
	} else {
		var d net.Dialer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err = d.DialContext(ctx, "tcp", address)
	}

	return
}

func (o *TCPOutput) String() string {
	return fmt.Sprintf("TCP output %s, limit: %d", o.address, o.limit)
}

func (o *TCPOutput) Close() {
	o.close = true
}
