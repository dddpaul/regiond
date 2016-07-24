package main

import (
	// "github.com/pkg/profile"
	"smilenet.ru/fedpa/cmd"
)

func main() {
	// defer profile.Start(profile.CPUProfile, profile.ProfilePath("pprof")).Stop()
	cmd.Execute()
}
