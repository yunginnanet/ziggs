package common

import (
	"os"
	"runtime/debug"
)

var Home string

func Version() (compileTime string, vcsRev string) {
	binInfo := make(map[string]string)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, v := range info.Settings {
		binInfo[v.Key] = v.Value
	}
	return binInfo["vcs.time"], binInfo["vcs.revision"]
}

const (
	// Title is the name of the application used throughout the configuration process.
	Title = "ziggs"
)

func init() {
	var err error
	if Home, err = os.UserHomeDir(); err != nil {
		panic(err)
	}
}
