package ziggy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"git.tcp.direct/kayos/common/entropy"
	"github.com/amimof/huego"
	tui "github.com/manifoldco/promptui"
	"github.com/rs/zerolog"
	"golang.org/x/net/proxy"

	"git.tcp.direct/kayos/ziggs/internal/common"
	"git.tcp.direct/kayos/ziggs/internal/config"
)

var log *zerolog.Logger

var errNoBridges = errors.New("no bridges available")

type Meta struct {
	Bridges  map[string]*Bridge
	Lights   map[string]*HueLight
	Switches map[string]*huego.Sensor
	Groups   map[string]*huego.Group
	*sync.RWMutex
}

// Lucifer is the lightbringer.
var Lucifer = Meta{
	Bridges:  make(map[string]*Bridge),
	Lights:   make(map[string]*HueLight),
	Switches: make(map[string]*huego.Sensor),
	RWMutex:  &sync.RWMutex{},
}

// Bridge represents a zigbee light controller. Just hue for now.
type Bridge struct {
	config    *config.KnownBridge
	Info      *huego.Config
	HueLights []*HueLight
	*huego.Bridge
	*sync.RWMutex
}

func (c *Bridge) Log() *zerolog.Logger {
	l := log.With().
		Str("caller", c.Info.BridgeID).
		Str("ip", c.Info.IPAddress).
		Uint8("zb", c.Info.ZigbeeChannel).Logger()
	return &l
}

type HueLight struct {
	huego.Light
	controller *Bridge
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
	return huego.NewWithClient(cridge.Hostname, cridge.Username, newClient)
}

func newController(cridge *config.KnownBridge) (*Bridge, error) {
	c := &Bridge{
		config:  cridge,
		RWMutex: &sync.RWMutex{},
	}
	if c.config.Proxy == "" {
		c.Bridge = huego.New(c.config.Hostname, c.config.Username)
	} else {
		c.Bridge = getProxiedBridge(cridge)
	}

	var err error
	c.Info, err = c.GetConfig()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func GetControllers(bridges []config.KnownBridge) (br []*Bridge) {
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

func (c *Bridge) getLights() error {
	var err error
	var l []huego.Light

	l, err = c.GetLights()
	if err != nil {
		return err
	}
	c.Log().Info().Msgf("Found %d lights", len(l))
	for _, light := range l {
		newlight := &HueLight{
			Light:      light,
			controller: c,
		}
		newlight.Log().Trace().Msg("+")
		c.HueLights = append(c.HueLights, newlight)
		Lucifer.Lock()
		name := strings.ReplaceAll(newlight.Name, " ", "_")
		if _, ok := Lucifer.Lights[name]; ok {
			name = fmt.Sprintf("%s_%d", name, 1)
		}
		Lucifer.Lights[name] = newlight
		Lucifer.Unlock()
	}
	return nil
}

func (c *Bridge) Lights() []*HueLight {
	if len(c.HueLights) > 0 {
		return c.HueLights
	}
	_ = c.getLights()
	return c.HueLights
}

func promptForUser(cnt *Bridge) bool {
	log.Info().Msg("found new bridge")
	confirmPrompt := tui.Select{
		Label:     "How should we authenticate?",
		Items:     []string{"Create new user", "Provide existing username"},
		CursorPos: 0,
		IsVimMode: false,
		Pointer:   common.ZiggsPointer,
	}
	choice, _, _ := confirmPrompt.Run()
	switch choice {
	case 0:
		println("press the link button on your bridge, then press enter")
		fmt.Scanln()
		newuser, err := cnt.CreateUser("ziggs" + strconv.Itoa(entropy.RNG(5)))
		if err != nil {
			log.Error().Err(err).Msg("failed")
			return false
		}
		cnt.User = newuser
	case 1:
		userEntry := tui.Prompt{
			Label: "Username:",
			Validate: func(s string) error {
				if len(s) < 40 {
					return errors.New("username must be at least 40 characters")
				}
				return nil
			},
			Mask:        'x',
			HideEntered: false,
			Pointer:     common.ZiggsPointer,
		}
		var err error
		var input string
		input, err = userEntry.Run()
		if err != nil {
			log.Error().Err(err).Msg("failed")
		}
		cnt.User = strings.TrimSpace(input)
	}
	log.Info().Str("caller", cnt.Host).Msg("logging in...")
	log.Trace().Msg("logging in using: " + cnt.User)
	cnt.Bridge = cnt.Bridge.Login(cnt.User)
	_, err := cnt.Bridge.GetCapabilities()
	if err != nil {
		log.Error().Err(err).Msg("failed to verify that we are logged in")
		return false
	}
	config.Snek.Set("bridges", map[string]interface{}{
		"hostname": cnt.Host,
		"username": cnt.User,
	})
	if err = config.Snek.WriteConfig(); err != nil {
		log.Warn().Msg("failed to write config")
	} else {
		log.Info().Msg("configuration saved!")
	}
	return true
}

func getBridgeInfo(c *Bridge) error {
	log.Trace().Msg("getting bridge config...")
	conf, err := c.GetConfig()
	if err != nil {
		return err
	}
	c.Info = conf
	c.config = &config.KnownBridge{
		Hostname: conf.IPAddress,
	}
	return c.getLights()
}

func Setup() (known []*Bridge, err error) {
	log = config.GetLogger()
	log.Debug().Int("count", len(config.KnownBridges)).Msg("trying bridges...")
	known = GetControllers(config.KnownBridges)
	if len(known) < 1 {
		err := promptForDiscovery()
		if err != nil {
			return []*Bridge{}, err
		}
		for _, cnt := range Lucifer.Bridges {
			cnt.RLock()
			log.Trace().Str("caller", cnt.Info.BridgeID).Int("lights", len(cnt.HueLights)).Msg("done")
			cnt.RUnlock()
		}
	}

	for _, bridge := range known {
		bridge.Log().Trace().Str("caller", bridge.ID).Str("mac", bridge.Info.Mac).Msg("getting lights..")
		err = bridge.getLights()
		if err != nil {
			bridge.Log().Warn().Err(err).Msg("failed to get lights")
			continue
		}
		var caps *huego.Capabilities
		caps, err = bridge.GetCapabilities()
		if err != nil {
			bridge.Log().Warn().Err(err).Msg("failed to get caps")
			continue
		}
		bridge.Log().Trace().Interface("supported", caps).Msg("capabilities")
		Lucifer.Lock()
		Lucifer.Bridges[bridge.Info.BridgeID] = bridge
		Lucifer.Unlock()
	}
	return
}
