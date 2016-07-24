package main

import (
	// "github.com/pkg/profile"

	// _ "net/http/pprof"

	"smilenet.ru/fedpa/cmd"
)

func main() {
	// Uncomment for CPU profiling (one must not use Oracle driver with this because of core dump)
	// defer profile.Start(profile.CPUProfile, profile.ProfilePath("pprof")).Stop()

	// Uncomment for any profiling (works OK with Oracle driver)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	cmd.Execute()
}
