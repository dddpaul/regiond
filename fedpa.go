package main

import (
	// "github.com/pkg/profile"

	_ "net/http/pprof"

	"smilenet.ru/fedpa/cmd"
)

func main() {
	// Uncomment for CPU profiling (one must not use Oracle driver with this because of core dump)
	// defer profile.Start(profile.CPUProfile, profile.ProfilePath("pprof")).Stop()

	cmd.Execute()
}
