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
	// GenConfig when toggled causes ziggs to write its default config to the cwd and then exit.
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
	// HTTPBind is defined via our toml configuration file. It is the address that ziggs listens on.
	HTTPBind string
	// HTTPPort is defined via our toml configuration file. It is the port that ziggs listens on.
	HTTPPort int
	// APIKey represents our key for API authentication.
	APIKey string
	// SSHListen is the address that ziggs listens on for SSH connections.
	SSHListen string
	// SSHHostKey is the path to the SSH host key, if any. If none is specified, one will be generated.
	SSHHostKey string
	// SSHPublicKeys is a list of public keys that are allowed to connect to ziggs via SSH.
	SSHPublicKeys []string
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
