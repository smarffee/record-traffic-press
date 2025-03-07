package settings

import (
	"net"
	"net/url"
	"record-traffic-press/goreplay/common"
	"record-traffic-press/goreplay/core/capture"
	"time"
)

// Settings holds Gor configuration
var Settings AppSettings

// AppSettings is the struct of main configuration
type AppSettings struct {
	Verbose   int           `json:"verbose"`
	Stats     bool          `json:"stats"`
	ExitAfter time.Duration `json:"exit-after"`

	Pprof string `json:"http-pprof"`

	CopyBufferSize common.Size `json:"copy-buffer-size"`

	InputDummy  []string `json:"input-dummy"`
	OutputDummy []string

	OutputStdout bool `json:"output-stdout"`
	OutputNull   bool `json:"output-null"`

	InputTCPConfig  TCPInputConfig
	OutputTCPConfig TCPOutputConfig

	InputTCP  []string `json:"input-tcp"`
	OutputTCP []string `json:"output-tcp"`

	OutputTCPStats bool `json:"output-tcp-stats"`

	OutputWebSocket       []string `json:"output-ws"`
	OutputWebSocketConfig WebSocketOutputConfig
	OutputWebSocketStats  bool `json:"output-ws-stats"`

	InputFile          []string      `json:"input-file"`
	InputFileLoop      bool          `json:"input-file-loop"`
	InputFileReadDepth int           `json:"input-file-read-depth"`
	InputFileDryRun    bool          `json:"input-file-dry-run"`
	InputFileMaxWait   time.Duration `json:"input-file-max-wait"`
	OutputFile         []string      `json:"output-file"`
	OutputFileConfig   FileOutputConfig

	InputRAW       []string `json:"input-raw"`
	InputRAWConfig RAWInputConfig

	Middleware string `json:"middleware"`

	InputHTTP    []string
	OutputHTTP   []string `json:"output-http"`
	PrettifyHTTP bool     `json:"prettify-http"`

	OutputHTTPConfig HTTPOutputConfig

	OutputBinary       []string `json:"output-binary"`
	OutputBinaryConfig BinaryOutputConfig

	ModifierConfig HTTPModifierConfig
}

// RAWInputConfig represents configuration that can be applied on raw input
type RAWInputConfig = capture.PcapOptions

// TCPInputConfig represents configuration of a TCP input plugin
type TCPInputConfig struct {
	Secure          bool   `json:"input-tcp-secure"`
	CertificatePath string `json:"input-tcp-certificate"`
	KeyPath         string `json:"input-tcp-certificate-key"`
}

// TCPOutputConfig tcp output configuration
type TCPOutputConfig struct {
	Secure     bool `json:"output-tcp-secure"`
	Sticky     bool `json:"output-tcp-sticky"`
	SkipVerify bool `json:"output-tcp-skip-verify"`
	Workers    int  `json:"output-tcp-workers"`

	GetInitMessage     func() *common.Message                         `json:"-"`
	WriteBeforeMessage func(conn net.Conn, msg *common.Message) error `json:"-"`
}

// FileOutputConfig ...
type FileOutputConfig struct {
	FlushInterval     time.Duration `json:"output-file-flush-interval"`
	SizeLimit         common.Size   `json:"output-file-size-limit"`
	OutputFileMaxSize common.Size   `json:"output-file-max-size-limit"`
	QueueLimit        int           `json:"output-file-queue-limit"`
	Append            bool          `json:"output-file-append"`
	BufferPath        string        `json:"output-file-buffer"`
	OnClose           func(string)  `json:"-"`
}

// WebSocketOutputConfig WebSocket output configuration
type WebSocketOutputConfig struct {
	Sticky     bool `json:"output-ws-sticky"`
	SkipVerify bool `json:"output-ws-skip-verify"`
	Workers    int  `json:"output-ws-workers"`

	Headers map[string][]string `json:"output-ws-headers"`
}

// BinaryOutputConfig struct for holding binary output configuration
type BinaryOutputConfig struct {
	Workers        int           `json:"output-binary-workers"`
	Timeout        time.Duration `json:"output-binary-timeout"`
	BufferSize     common.Size   `json:"output-tcp-response-buffer"`
	Debug          bool          `json:"output-binary-debug"`
	TrackResponses bool          `json:"output-binary-track-response"`
}

// HTTPOutputConfig struct for holding http output configuration
type HTTPOutputConfig struct {
	TrackResponses    bool          `json:"output-http-track-response"`
	Stats             bool          `json:"output-http-stats"`
	OriginalHost      bool          `json:"output-http-original-host"`
	RedirectLimit     int           `json:"output-http-redirect-limit"`
	WorkersMin        int           `json:"output-http-workers-min"`
	WorkersMax        int           `json:"output-http-workers"`
	StatsMs           int           `json:"output-http-stats-ms"`
	QueueLen          int           `json:"output-http-queue-len"`
	ElasticSearch     string        `json:"output-http-elasticsearch"`
	Timeout           time.Duration `json:"output-http-timeout"`
	WorkerTimeout     time.Duration `json:"output-http-worker-timeout"`
	BufferSize        common.Size   `json:"output-http-response-buffer"`
	SkipVerify        bool          `json:"output-http-skip-verify"`
	CompatibilityMode bool          `json:"output-http-compatibility-mode"`
	RequestGroup      string        `json:"output-http-request-group"`
	Debug             bool          `json:"output-http-debug"`
	RawURL            string        `json:"-"`
	Url               *url.URL      `json:"-"`
}

func (hoc *HTTPOutputConfig) Copy() *HTTPOutputConfig {
	return &HTTPOutputConfig{
		TrackResponses:    hoc.TrackResponses,
		Stats:             hoc.Stats,
		OriginalHost:      hoc.OriginalHost,
		RedirectLimit:     hoc.RedirectLimit,
		WorkersMin:        hoc.WorkersMin,
		WorkersMax:        hoc.WorkersMax,
		StatsMs:           hoc.StatsMs,
		QueueLen:          hoc.QueueLen,
		ElasticSearch:     hoc.ElasticSearch,
		Timeout:           hoc.Timeout,
		WorkerTimeout:     hoc.WorkerTimeout,
		BufferSize:        hoc.BufferSize,
		SkipVerify:        hoc.SkipVerify,
		CompatibilityMode: hoc.CompatibilityMode,
		RequestGroup:      hoc.RequestGroup,
		Debug:             hoc.Debug,
	}
}

func CheckSettings() {

	if Settings.OutputFileConfig.SizeLimit < 1 {
		Settings.OutputFileConfig.SizeLimit.Set("32mb")
	}
	if Settings.OutputFileConfig.OutputFileMaxSize < 1 {
		Settings.OutputFileConfig.OutputFileMaxSize.Set("1tb")
	}
	if Settings.CopyBufferSize < 1 {
		Settings.CopyBufferSize.Set("5mb")
	}
}
