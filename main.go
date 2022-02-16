package main

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/amimof/huego"
	"github.com/rs/zerolog"
	"golang.org/x/net/proxy"

	"git.tcp.direct/kayos/ziggs/config"
)

var (
	log *zerolog.Logger
)

var Known = make(map[string]*Controller)
var AllLights []*Light

// Controller is just another word for a bridge, a light controller.
type Controller struct {
	config *config.KnownBridge
	info   *huego.Config
	lights []*Light
	*huego.Bridge
	log zerolog.Logger
}

type Light struct {
	*huego.Light
	log        zerolog.Logger
	controller *Controller
}

func getProxiedBridge(cridge *config.KnownBridge) *huego.Bridge {
	cridge.Proxy = strings.TrimPrefix(cridge.Proxy, "socks5://")
	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	proxyDialer, _ := proxy.SOCKS5("tcp", cridge.Proxy, nil, proxy.Direct)
	newTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return proxyDialer.Dial(network, addr)
	}
	newClient := http.DefaultClient
	newClient.Transport = newTransport
	return huego.NewCustom(cridge.Hostname, cridge.Username, newClient)
}

func newController(cridge *config.KnownBridge) (*Controller, error) {
	c := &Controller{
		config: cridge,
	}
	if c.config.Proxy == "" {
		c.Bridge = huego.New(c.config.Hostname, c.config.Username)
	} else {
		c.Bridge = getProxiedBridge(cridge)
	}

	var err error
	c.info, err = c.GetConfig()
	if err != nil {
		return nil, err
	}
	c.log = log.With().Str("caller", c.info.BridgeID).Str("mac", c.info.Mac).Logger()
	return c, nil
}

func GetControllers(bridges []config.KnownBridge) {
	for _, lightConfig := range bridges {
		log.Debug().Str("caller", lightConfig.Hostname).Str("proxy", lightConfig.Proxy).Msg("attempting connection")
		c, err := newController(&lightConfig)
		if err != nil {
			log.Error().Str("caller", lightConfig.Hostname).Err(err).Msg("unsuccessful connection")
			continue
		}
		c.log.Info().Msg("successful connection")
		Known[c.info.BridgeID] = c
	}
}

type ToggleMode uint8

const (
	ToggleOn ToggleMode = iota
	ToggleOff
	Toggle
)

func toggle(l *Light, mode ToggleMode) error {
	on := func(l *Light) error {
		l.log.Trace().Msg("turning light on...")
		return l.On()
	}
	off := func(l *Light) error {
		l.log.Trace().Msg("turning light off...")
		return l.Off()
	}
	var err error
	switch mode {
	case ToggleOn:
		err = on(l)
	case ToggleOff:
		err = off(l)
	case Toggle:
		if !l.IsOn() {
			err = off(l)
			break
		}
		err = on(l)
	default:
		//
	}
	return err
}

func ToggleLights(Lights []*Light, mode ToggleMode) {
	for _, l := range Lights {
		err := toggle(l, mode)
		if err != nil {
			l.log.Error().Err(err).Bool("On", l.IsOn()).Msg("failed to toggle light")
		}
	}
}

func Setup() {
	log.Debug().Int("count", len(config.KnownBridges)).Msg("trying bridges...")
	GetControllers(config.KnownBridges)
	if len(Known) == 0 {
		log.Fatal().Msg("failed to connect to any bridges")
	}

	for _, bridge := range Known {
		l, err := bridge.GetLights()
		if err != nil || len(l) < 1 {
			log.Fatal().Err(err).Msg("failed to discover lights")
		}
		bridge.log.Info().Msgf("Found %d lights", len(l))
		for _, light := range l {
			newlight := &Light{
				Light: &light,
				log: bridge.log.With().
					Int("caller", light.ID).
					Str("name", light.Name).Logger(),
				controller: bridge,
			}
			bridge.lights = append(bridge.lights, newlight)
			AllLights = append(AllLights, newlight)
		}
	}
}

func init() {
	config.Init()
	log = config.StartLogger()
}

func main() {
	Setup()
	for name, bridge := range Known {
		bridge.log.Debug().Str("caller", name).Interface("details", bridge).Msg("+")
		for _, l := range bridge.lights {
			l.log.Debug().Str("caller", bridge.ID).Msg("+")
		}
	}
}
