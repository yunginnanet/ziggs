package common

import "os"

var (
	Home string
)

const (
	// Version roughly represents the applications current version.
	Version = "0.1"
	// Title is the name of the application used throughout the configuration process.
	Title = "ziggs"
)

func init() {
	var err error
	if Home, err = os.UserHomeDir(); err != nil {
		panic(err)
	}
}
