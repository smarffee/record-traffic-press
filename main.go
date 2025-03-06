// Gor is simple http traffic replication tool written in Go. Its main goal to replay traffic from production servers to staging and dev environments.
// Now you can test your code on real user sessions in an automated and repeatable fashion.
package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	httppptof "net/http/pprof"
	"os"
	"os/signal"
	"record-traffic-press/goreplay/bootstrap"
	"record-traffic-press/goreplay/core"
	"record-traffic-press/goreplay/input"
	"record-traffic-press/goreplay/output"
	"record-traffic-press/goreplay/settings"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to this file")
)

func init() {
	var defaultServeMux http.ServeMux
	http.DefaultServeMux = &defaultServeMux

	http.HandleFunc("/debug/vars", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "{\n")
		first := true
		expvar.Do(func(kv expvar.KeyValue) {
			if kv.Key == "memstats" || kv.Key == "cmdline" {
				return
			}

			if !first {
				fmt.Fprintf(w, ",\n")
			}
			first = false
			fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
		})
		fmt.Fprintf(w, "\n}\n")
	})

	http.HandleFunc("/debug/pprof/", httppptof.Index)
	http.HandleFunc("/debug/pprof/cmdline", httppptof.Cmdline)
	http.HandleFunc("/debug/pprof/profile", httppptof.Profile)
	http.HandleFunc("/debug/pprof/symbol", httppptof.Symbol)
	http.HandleFunc("/debug/pprof/trace", httppptof.Trace)
}

func loggingMiddleware(addr string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/loop" {
			_, err := http.Get("http://" + addr)
			log.Println(err)
		}

		rb, _ := httputil.DumpRequest(r, false)
		log.Println(string(rb))
		next.ServeHTTP(w, r)
	})
}

func main() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	}

	settings.CheckSettings()

	p := NewPlugins()

	log.Printf("[PPID %d and PID %d] Version:%s\n", os.Getppid(), os.Getpid(), bootstrap.VERSION)

	if len(p.Inputs) == 0 || len(p.Outputs) == 0 {
		log.Fatal("Required at least 1 input and 1 output")
	}

	if *memprofile != "" {
		profileMEM(*memprofile)
	}

	if *cpuprofile != "" {
		profileCPU(*cpuprofile)
	}

	if settings.Settings.Pprof != "" {
		go func() {
			log.Println(http.ListenAndServe(settings.Settings.Pprof, nil))
		}()
	}

	closeCh := make(chan int)
	emitter := bootstrap.NewEmitter()

	go emitter.Start(p, settings.Settings.Middleware)
	if settings.Settings.ExitAfter > 0 {
		log.Printf("Running gor for a duration of %s\n", settings.Settings.ExitAfter)

		time.AfterFunc(settings.Settings.ExitAfter, func() {
			log.Printf("gor run timeout %s\n", settings.Settings.ExitAfter)
			close(closeCh)
		})
	}

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	exit := 0
	select {
	case <-c:
		exit = 1
	case <-closeCh:
		exit = 0
	}

	emitter.Close()
	os.Exit(exit)
}

func profileCPU(cpuprofile string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)

		time.AfterFunc(30*time.Second, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}
}

func profileMEM(memprofile string) {
	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal(err)
		}
		time.AfterFunc(30*time.Second, func() {
			pprof.WriteHeapProfile(f)
			f.Close()
		})
	}
}

// NewPlugins specify and initialize all available plugins
func NewPlugins() *core.InOutPlugins {
	plugins := new(core.InOutPlugins)

	for _, options := range settings.Settings.InputDummy {
		plugins.RegisterPlugin(input.NewDummyInput, options)
	}

	for range settings.Settings.OutputDummy {
		plugins.RegisterPlugin(output.NewDummyOutput)
	}

	if settings.Settings.OutputStdout {
		plugins.RegisterPlugin(output.NewDummyOutput)
	}

	if settings.Settings.OutputNull {
		plugins.RegisterPlugin(output.NewNullOutput)
	}

	for _, options := range settings.Settings.InputRAW {
		plugins.RegisterPlugin(input.NewRAWInput, options, settings.Settings.InputRAWConfig)
	}

	for _, options := range settings.Settings.InputTCP {
		plugins.RegisterPlugin(input.NewTCPInput, options, &settings.Settings.InputTCPConfig)
	}

	for _, options := range settings.Settings.OutputTCP {
		plugins.RegisterPlugin(output.NewTCPOutput, options, &settings.Settings.OutputTCPConfig)
	}

	for _, options := range settings.Settings.OutputWebSocket {
		plugins.RegisterPlugin(output.NewWebSocketOutput, options, &settings.Settings.OutputWebSocketConfig)
	}

	for _, options := range settings.Settings.InputFile {
		plugins.RegisterPlugin(input.NewFileInput, options, settings.Settings.InputFileLoop, settings.Settings.InputFileReadDepth, settings.Settings.InputFileMaxWait, settings.Settings.InputFileDryRun)
	}

	for _, path := range settings.Settings.OutputFile {
		plugins.RegisterPlugin(output.NewFileOutput, path, &settings.Settings.OutputFileConfig)
	}

	for _, options := range settings.Settings.InputHTTP {
		plugins.RegisterPlugin(input.NewHTTPInput, options)
	}

	// If we explicitly set Host header http output should not rewrite it
	// Fix: https://record-traffic-press/gor/issues/174
	for _, header := range settings.Settings.ModifierConfig.Headers {
		if header.Name == "Host" {
			settings.Settings.OutputHTTPConfig.OriginalHost = true
			break
		}
	}

	for _, options := range settings.Settings.OutputHTTP {
		plugins.RegisterPlugin(output.NewHTTPOutput, options, &settings.Settings.OutputHTTPConfig)
	}

	for _, options := range settings.Settings.OutputBinary {
		plugins.RegisterPlugin(output.NewBinaryOutput, options, &settings.Settings.OutputBinaryConfig)
	}

	return plugins
}
