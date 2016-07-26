package main

import (
	// "github.com/pkg/profile"
	_ "expvar" // Export metrics
	"net/http"
	_ "net/http/pprof" // HTTP profiling

	"smilenet.ru/fedpa/cmd"
)

func main() {
	// Uncomment for CPU profiling (one must not use Oracle driver with this because of core dump)
	// defer profile.Start(profile.CPUProfile, profile.ProfilePath("pprof")).Stop()

	// HTTP server for metrics and profiling
	go http.ListenAndServe(":8123", nil)

	cmd.Execute()
}
