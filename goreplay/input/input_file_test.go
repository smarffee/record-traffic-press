package input

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"record-traffic-press/goreplay"
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/core"
	output3 "record-traffic-press/goreplay/output"
	"record-traffic-press/goreplay/plugins"
	"record-traffic-press/goreplay/plugins/output"
	"record-traffic-press/goreplay/settings"
	"sync"
	"testing"
	"time"
)

func TestInputFileWithGET(t *testing.T) {
	input := bootstrap.NewTestInput()
	rg := NewRequestGenerator([]core.PluginReader{input}, func() { input.EmitGET() }, 1)
	readPayloads := []*plugins.Message{}

	// Given a capture file with a GET request
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the request is read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 1, func(msg *plugins.Message) {
		readPayloads = append(readPayloads, msg)
	})

	// The read request should match the original request
	if err != nil {
		t.Error(err)
	} else if !expectedCaptureFile.PayloadsEqual(readPayloads) {
		t.Error("Request read back from file should match")

	}
}

func TestInputFileWithPayloadLargerThan64Kb(t *testing.T) {
	input := bootstrap.NewTestInput()
	rg := NewRequestGenerator([]core.PluginReader{input}, func() { input.EmitSizedPOST(64 * 1024) }, 1)
	readPayloads := []*plugins.Message{}

	// Given a capture file with a request over 64Kb
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the request is read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 1, func(msg *plugins.Message) {
		readPayloads = append(readPayloads, msg)
	})

	// The read request should match the original request
	if err != nil {
		t.Error(err)
	} else if !expectedCaptureFile.PayloadsEqual(readPayloads) {
		t.Error("Request read back from file should match")

	}

}

func TestInputFileWithGETAndPOST(t *testing.T) {

	input := bootstrap.NewTestInput()
	rg := NewRequestGenerator([]core.PluginReader{input}, func() {
		input.EmitGET()
		input.EmitPOST()
	}, 2)
	readPayloads := []*plugins.Message{}

	// Given a capture file with a GET request
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the requests are read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 2, func(msg *plugins.Message) {
		readPayloads = append(readPayloads, msg)
	})

	// The read requests should match the original request
	if err != nil {
		t.Error(err)
	} else if !expectedCaptureFile.PayloadsEqual(readPayloads) {
		t.Error("Request read back from file should match")

	}

}

func TestInputFileMultipleFilesWithRequestsOnly(t *testing.T) {
	rnd := rand.Int63()

	file1, _ := os.OpenFile(fmt.Sprintf("/tmp/%d_0", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	file1.Write([]byte("1 1 1\ntest1"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Write([]byte("1 1 3\ntest2"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Close()

	file2, _ := os.OpenFile(fmt.Sprintf("/tmp/%d_1", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	file2.Write([]byte("1 1 2\ntest3"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Write([]byte("1 1 4\ntest4"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Close()

	input := NewFileInput(fmt.Sprintf("/tmp/%d*", rnd), false, 100, 0, false)

	for i := '1'; i <= '4'; i++ {
		msg, _ := input.PluginRead()
		if msg.Meta[4] != byte(i) {
			t.Error("Should emit requests in right order", string(msg.Meta))
		}
	}

	os.Remove(file1.Name())
	os.Remove(file2.Name())
}

func TestInputFileRequestsWithLatency(t *testing.T) {
	rnd := rand.Int63()

	file, _ := os.OpenFile(fmt.Sprintf("/tmp/%d", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	defer file.Close()

	file.Write([]byte("1 1 100000000\nrequest1"))
	file.Write([]byte(goreplay.payloadSeparator))
	file.Write([]byte("1 2 150000000\nrequest2"))
	file.Write([]byte(goreplay.payloadSeparator))
	file.Write([]byte("1 3 250000000\nrequest3"))
	file.Write([]byte(goreplay.payloadSeparator))

	input := NewFileInput(fmt.Sprintf("/tmp/%d", rnd), false, 100, 0, false)

	start := time.Now().UnixNano()
	for i := 0; i < 3; i++ {
		input.PluginRead()
	}
	end := time.Now().UnixNano()

	var expectedLatency int64 = 300000000 - 100000000
	realLatency := end - start
	if realLatency > expectedLatency {
		t.Errorf("Should emit requests respecting latency. Expected: %v, real: %v", expectedLatency, realLatency)
	}
}

func TestInputFileMultipleFilesWithRequestsAndResponses(t *testing.T) {
	rnd := rand.Int63()

	file1, _ := os.OpenFile(fmt.Sprintf("/tmp/%d_0", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	file1.Write([]byte("1 1 1\nrequest1"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Write([]byte("2 1 1\nresponse1"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Write([]byte("1 2 3\nrequest2"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Write([]byte("2 2 3\nresponse2"))
	file1.Write([]byte(goreplay.payloadSeparator))
	file1.Close()

	file2, _ := os.OpenFile(fmt.Sprintf("/tmp/%d_1", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	file2.Write([]byte("1 3 2\nrequest3"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Write([]byte("2 3 2\nresponse3"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Write([]byte("1 4 4\nrequest4"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Write([]byte("2 4 4\nresponse4"))
	file2.Write([]byte(goreplay.payloadSeparator))
	file2.Close()

	input := NewFileInput(fmt.Sprintf("/tmp/%d*", rnd), false, 100, 0, false)

	for i := '1'; i <= '4'; i++ {
		msg, _ := input.PluginRead()
		if msg.Meta[0] != '1' && msg.Meta[4] != byte(i) {
			t.Error("Shound emit requests in right order", string(msg.Meta))
		}

		msg, _ = input.PluginRead()
		if msg.Meta[0] != '2' && msg.Meta[4] != byte(i) {
			t.Error("Shound emit responses in right order", string(msg.Meta))
		}
	}

	os.Remove(file1.Name())
	os.Remove(file2.Name())
}

func TestInputFileLoop(t *testing.T) {
	rnd := rand.Int63()

	file, _ := os.OpenFile(fmt.Sprintf("/tmp/%d", rnd), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	file.Write([]byte("1 1 1\ntest1"))
	file.Write([]byte(goreplay.payloadSeparator))
	file.Write([]byte("1 1 2\ntest2"))
	file.Write([]byte(goreplay.payloadSeparator))
	file.Close()

	input := NewFileInput(fmt.Sprintf("/tmp/%d", rnd), true, 100, 0, false)

	// Even if we have just 2 requests in file, it should indifinitly loop
	for i := 0; i < 1000; i++ {
		input.PluginRead()
	}

	input.Close()
	os.Remove(file.Name())
}

func TestInputFileCompressed(t *testing.T) {
	rnd := rand.Int63()

	output := output3.NewFileOutput(fmt.Sprintf("/tmp/%d_0.gz", rnd), &output.FileOutputConfig{FlushInterval: time.Minute, Append: true})
	for i := 0; i < 1000; i++ {
		output.PluginWrite(&plugins.Message{Meta: []byte("1 1 1\r\n"), Data: []byte("test")})
	}
	name1 := output.file.Name()
	output.Close()

	output2 := output.NewFileOutput(fmt.Sprintf("/tmp/%d_1.gz", rnd), &output.FileOutputConfig{FlushInterval: time.Minute, Append: true})
	for i := 0; i < 1000; i++ {
		output2.PluginWrite(&plugins.Message{Meta: []byte("1 1 1\r\n"), Data: []byte("test")})
	}
	name2 := output2.file.Name()
	output2.Close()

	input := NewFileInput(fmt.Sprintf("/tmp/%d*", rnd), false, 100, 0, false)
	for i := 0; i < 2000; i++ {
		input.PluginRead()
	}

	os.Remove(name1)
	os.Remove(name2)
}

type CaptureFile struct {
	msgs []*plugins.Message
	file *os.File
}

func NewExpectedCaptureFile(msgs []*plugins.Message, file *os.File) *CaptureFile {
	ecf := new(CaptureFile)
	ecf.file = file
	ecf.msgs = msgs
	return ecf
}

func (expectedCaptureFile *CaptureFile) TearDown() {
	if expectedCaptureFile.file != nil {
		os.Remove(expectedCaptureFile.file.Name())
	}
}

type RequestGenerator struct {
	inputs []core.PluginReader
	emit   func()
	wg     *sync.WaitGroup
}

func NewRequestGenerator(inputs []core.PluginReader, emit func(), count int) (rg *RequestGenerator) {
	rg = new(RequestGenerator)
	rg.inputs = inputs
	rg.emit = emit
	rg.wg = new(sync.WaitGroup)
	rg.wg.Add(count)
	return
}

func (expectedCaptureFile *CaptureFile) PayloadsEqual(other []*plugins.Message) bool {

	if len(expectedCaptureFile.msgs) != len(other) {
		return false
	}

	for i, payload := range other {
		if !bytes.Equal(expectedCaptureFile.msgs[i].Meta, payload.Meta) {
			return false
		}
		if !bytes.Equal(expectedCaptureFile.msgs[i].Data, payload.Data) {
			return false
		}
	}

	return true

}

func CreateCaptureFile(requestGenerator *RequestGenerator) *CaptureFile {
	f, err := ioutil.TempFile("", "testmainconf")
	if err != nil {
		panic(err)
	}

	readPayloads := []*plugins.Message{}
	output := bootstrap.NewTestOutput(func(msg *plugins.Message) {
		readPayloads = append(readPayloads, msg)
		requestGenerator.wg.Done()
	})

	outputFile := output.NewFileOutput(f.Name(), &output.FileOutputConfig{FlushInterval: time.Second, Append: true})

	plugins := &core.InOutPlugins{
		Inputs:  requestGenerator.inputs,
		Outputs: []core.PluginWriter{output, outputFile},
	}
	for _, input := range requestGenerator.inputs {
		plugins.All = append(plugins.All, input)
	}
	plugins.All = append(plugins.All, output, outputFile)

	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	requestGenerator.emit()
	requestGenerator.wg.Wait()

	time.Sleep(100 * time.Millisecond)
	emitter.Close()

	return NewExpectedCaptureFile(readPayloads, f)

}

func ReadFromCaptureFile(captureFile *os.File, count int, callback goreplay.writeCallback) (err error) {
	wg := new(sync.WaitGroup)

	input := NewFileInput(captureFile.Name(), false, 100, 0, false)
	output := bootstrap.NewTestOutput(func(msg *plugins.Message) {
		callback(msg)
		wg.Done()
	})

	plugins := &core.InOutPlugins{
		Inputs:  []core.PluginReader{input},
		Outputs: []core.PluginWriter{output},
	}
	plugins.All = append(plugins.All, input, output)

	wg.Add(count)
	emitter := bootstrap.NewEmitter()
	go emitter.Start(plugins, settings.Settings.Middleware)

	done := make(chan int, 1)
	go func() {
		wg.Wait()
		done <- 1
	}()

	select {
	case <-done:
		break
	case <-time.After(2 * time.Second):
		err = errors.New("Timed out")
	}
	emitter.Close()
	return

}
