package lights

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/amimof/huego"
	"github.com/rs/zerolog"
	"golang.org/x/net/proxy"

	"git.tcp.direct/kayos/ziggs/config"
)

var log *zerolog.Logger

// Controller is just another word for a bridge, a light controller.
type Controller struct {
	config    *config.KnownBridge
	info      *huego.Config
	HueLights []*HueLight
	*huego.Bridge
}

func (c *Controller) Log() *zerolog.Logger {
	l := log.With().
		Str("caller", c.info.BridgeID).
		Str("ip", c.info.IPAddress).
		Uint8("zb", c.info.ZigbeeChannel).Logger()
	return &l
}

type HueLight struct {
	huego.Light
	controller *Controller
}

func (hl *HueLight) Log() *zerolog.Logger {
	l := log.With().
		Int("caller", hl.ID).
		Str("name", hl.Name).
		Bool("on", hl.IsOn()).Logger()
	return &l
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

	return c, nil
}

func GetControllers(bridges []config.KnownBridge) (br []*Controller) {
	for _, lightConfig := range bridges {
		log.Debug().Str("caller", lightConfig.Hostname).Str("proxy", lightConfig.Proxy).Msg("attempting connection")
		c, err := newController(&lightConfig)
		if err != nil {
			log.Error().Str("caller", lightConfig.Hostname).Err(err).Msg("unsuccessful connection")
			continue
		}
		c.Log().Info().Msg("successful connection")
		br = append(br, c)
	}
	return
}

type ToggleMode uint8

const (
	ToggleOn ToggleMode = iota
	ToggleOff
	ToggleRainbow
	ToggleDim
	Toggle
)

type lCall func(light *HueLight) (checkFunc, error)
type checkFunc func(light *HueLight) bool

var lightCallbacks = map[ToggleMode]lCall{
	ToggleOn: func(light *HueLight) (checkFunc, error) {
		return func(light *HueLight) bool {
				light.State = &huego.State{
					On:     true,
					Bri:    100,
					Effect: "none",
					Scene:  "none",
				}
				if !light.IsOn() {
					return false
				}
				return light.IsOn()
			},
			light.On()
	},
	ToggleOff: func(light *HueLight) (checkFunc, error) {
		return func(light *HueLight) bool {
				light.State = &huego.State{
					On:     false,
					Bri:    100,
					Effect: "none",
					Scene:  "none",
				}
				if light.IsOn() {
					return false
				}
				return !light.IsOn()
			},
			light.Off()
	},
	/*	ToggleDim: func(light *HueLight) (checkFunc, error) {
		return func(light *HueLight) bool {
				if !light.IsOn() {
					return false
				}
				if light.State.Bri
			},
			light.On()
	},*/
	ToggleRainbow: func(light *HueLight) (checkFunc, error) {
		return func(light *HueLight) bool {
				if !light.IsOn() {
					return false
				}
				return light.State.Effect == "colorloop"
			},
			light.Effect("colorloop")
	},
}

func Assert(ctx context.Context, l *HueLight, mode ToggleMode) error {
	act, ok := lightCallbacks[mode]
	if !ok {
		panic("not implemented")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			check, err := act(l)
			if err != nil && !check(l) {
				log.Warn().Err(err).Msg("fail")
				continue
			}
			if !check(l) {
				continue
			}
			return nil
		}
	}
}

func toggle(l *HueLight, mode ToggleMode) error {
	on := func(l *HueLight) error {
		l.Log().Trace().Msg("turning light on...")
		return l.On()
	}
	off := func(l *HueLight) error {
		l.Log().Trace().Msg("turning light off...")
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

func ToggleLights(Lights []*HueLight, mode ToggleMode) {
	for _, l := range Lights {
		err := toggle(l, mode)
		if err != nil {
			l.Log().Error().Err(err).Bool("On", l.IsOn()).Msg("failed to toggle light")
		}
	}
}

func (c *Controller) getLights() error {
	var err error
	var l []huego.Light
	l, err = c.GetLights()
	if err != nil {
		return err
	}
	c.Log().Trace().Msgf("Found %d lights", len(l))
	for _, light := range l {
		newlight := &HueLight{
			Light:      light,
			controller: c,
		}
		c.HueLights = append(c.HueLights, newlight)
		newlight.SwConfigID

	}
	return nil
}

func (c *Controller) Lights() []*HueLight {
	if len(c.HueLights) > 0 {
		return c.HueLights
	} else {
		_ = c.getLights()
	}
	return c.HueLights
}

func Setup() (known []*Controller, err error) {
	log = config.GetLogger()
	log.Debug().Int("count", len(config.KnownBridges)).Msg("trying bridges...")
	known = GetControllers(config.KnownBridges)
	if len(known) == 0 {
		err = errors.New("no bridges connected successfully")
		return
	}

	for _, bridge := range known {
		log.Trace().Str("caller", bridge.ID).Str("mac", bridge.info.Mac).Msg("getting lights..")
		lerr := bridge.getLights()
		if lerr != nil {
			return known, lerr
		}
	}
	return
}
