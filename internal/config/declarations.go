package config

import (
	"os"

	"github.com/spf13/viper"
)

var (
	f   *os.File
	err error
)

var (
	customconfig    = false
	configLocations []string
)

var (
	// GenConfig when toggled causes HellPot to write its default config to the cwd and then exit.
	GenConfig = false
	// NoColor stops zerolog from outputting color, necessary on Windows.
	NoColor = true
)

// ----------------- //

// "lights"

// KnownBridge represents the part of our configuration that defines hue bridges to connect to.
type KnownBridge struct {
	Hostname string `mapstructure:"hostname"`
	Username string `mapstructure:"username"`
	Proxy    string `mapstructure:"proxy"`
}

// KnownBridges contains all of the bridges we already knew about from our config file.
var KnownBridges []KnownBridge

// "http"
var (
	// HTTPBind is defined via our toml configuration file. It is the address that HellPot listens on.
	HTTPBind string
	// HTTPPort is defined via our toml configuration file. It is the port that HellPot listens on.
	HTTPPort int
	// APIKey represents our key for API authentication.
	APIKey string
)

var (
	Debug bool
	Trace bool
	// Filename identifies the location of our configuration file.
	Filename           string
	prefConfigLocation string
	// Snek represents our instance of Viper.
	Snek *viper.Viper
)
