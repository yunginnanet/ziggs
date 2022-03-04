package lights

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

	"git.tcp.direct/kayos/ziggs/config"
)

var log *zerolog.Logger

var errNoBridges = errors.New("no bridges available")

type Meta struct {
	Bridges  map[string]*Bridge
	Lights   map[string]*HueLight
	Switches map[string]*huego.Sensor
	*sync.RWMutex
}

// Lucifer is the lightbringer.
var Lucifer = Meta{
	Bridges:  make(map[string]*Bridge),
	Lights:   make(map[string]*HueLight),
	Switches: make(map[string]*huego.Sensor),
	RWMutex:  &sync.RWMutex{},
}

// Bridge is just another word for a bridge, a light controller.
type Bridge struct {
	config    *config.KnownBridge
	info      *huego.Config
	HueLights []*HueLight
	*huego.Bridge
	*sync.RWMutex
}

func (c *Bridge) Log() *zerolog.Logger {
	l := log.With().
		Str("caller", c.info.BridgeID).
		Str("ip", c.info.IPAddress).
		Uint8("zb", c.info.ZigbeeChannel).Logger()
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

	defer c.Unlock()

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
		Lucifer.Lights[light.UniqueID] = newlight
		Lucifer.Unlock()
	}
	return nil
}

func (c *Bridge) Lights() []*HueLight {
	if len(c.HueLights) > 0 {
		return c.HueLights
	} else {
		_ = c.getLights()
	}
	return c.HueLights
}

func promptForUser(cnt *Bridge) bool {
	log.Info().Msg("found new bridge")
	confirmPrompt := tui.Select{
		Label:     "Create new user?",
		Items:     []string{"Yes", "No"},
		CursorPos: 0,
		IsVimMode: false,
		Pointer: func(x []rune) []rune {
			return []rune("")
		},
	}
	_, choice, _ := confirmPrompt.Run()
	if choice != "Yes" {
		println("press the link button on your bridge, then press enter")
		fmt.Scanln()
		newuser, err := cnt.CreateUser("ziggs" + strconv.Itoa(entropy.RNG(5)))
		if err != nil {
			log.Error().Err(err).Msg("failed")
			return false
		}
		log.Info().Str("caller", cnt.Host).Msg(newuser)
		log.Trace().Msg("logging in using: " + newuser)
		cnt.Bridge = cnt.Bridge.Login(newuser)
	}
}

func promptForDiscovery() error {
	log.Warn().Msg("failed to connect to known bridges from configuration file.")
	confirmPrompt := tui.Select{
		Label:     "Search for bridges?",
		Items:     []string{"Yes", "No"},
		CursorPos: 0,
		IsVimMode: false,
		Pointer: func(x []rune) []rune {
			return []rune("")
		},
	}
	_, choice, _ := confirmPrompt.Run()
	if choice != "Yes" {
		return errNoBridges
	}
	log.Info().Msg("searching for bridges...")
	cs, err := huego.DiscoverAll()
	if err != nil {
		return err
	}
	if len(cs) < 1 {
		return errNoBridges
	}
	for _, c := range cs {
		Lucifer.Lock()
		cnt := &Bridge{
			Bridge:  &c,
			RWMutex: &sync.RWMutex{},
		}
		Lucifer.Bridges[c.Host] = cnt
		Lucifer.Unlock()
		if promptForUser(cnt) {
			getBridgeInfo(cnt)
		}
	}
	return nil
}

func getBridgeInfo(c *Bridge) {
	conf, err := c.GetConfig()
	if err != nil {
		log.Warn().Err(err).Msg("failed to get config")
		return
	}
	c.info = conf
	c.config = &config.KnownBridge{
		Hostname: conf.IPAddress,
	}
	c.Lock()
	go c.getLights()
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
			log.Trace().Str("caller", cnt.info.BridgeID).Int("lights", len(cnt.HueLights)).Msg("done")
			cnt.RUnlock()
		}
	}

	for _, bridge := range known {
		bridge.Log().Trace().Str("caller", bridge.ID).Str("mac", bridge.info.Mac).Msg("getting lights..")
		err = bridge.getLights()
		if err != nil {
			return
		}
		var caps *huego.Capabilities
		caps, err = bridge.GetCapabilities()
		if err != nil {
			return
		}
		bridge.Log().Trace().Interface("supported", caps).Msg("capabilities")

	}
	return
}
