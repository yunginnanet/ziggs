package ziggy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
	"git.tcp.direct/kayos/common/network"
	"github.com/amimof/huego"
	tui "github.com/manifoldco/promptui"
	"github.com/rs/zerolog"
	"golang.org/x/net/proxy"
	"inet.af/netaddr"

	"git.tcp.direct/kayos/ziggs/common"
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

func filterCandidateInterfaces(interfaces []net.Interface) []net.Interface {
	var err error
	var candidates []net.Interface
addrIter:
	for _, iface := range interfaces {
		switch {
		case iface.Flags&net.FlagUp == 0,
			iface.Flags&net.FlagLoopback != 0,
			iface.Flags&net.FlagPointToPoint != 0,
			iface.HardwareAddr == nil:
			log.Debug().Msgf("skipping %s", iface.Name)
			continue
		default:
			var addrs []net.Addr
			addrs, err = iface.Addrs()
			if err != nil {
				log.Debug().Err(err).Msg("failed to get addresses")
				continue
			}
			for _, a := range addrs {
				ip := net.ParseIP(a.String())
				if ip != nil && !ip.IsPrivate() {
					log.Debug().Msgf("skipping interface %s with public IP: %s", iface.Name, ip)
					continue addrIter
				}
			}
			candidates = append(candidates, iface)
		}
	}
	return candidates
}

func enumerateBridge(a net.Addr) interface{} {
	var err error
	if _, err = net.DialTimeout("tcp", a.String()+":80", 2*time.Second); err != nil {
		log.Debug().Err(err).Msgf("failed to dial %s", a.String())
		return nil
	}
	var resp *http.Response
	c := http.DefaultClient
	c.Timeout = 2 * time.Second
	resp, err = c.Get("http://" + a.String() + "/api/config")
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get %s", a.String())
		return nil
	}
	if resp.StatusCode != 200 {
		log.Debug().Msgf("%s returned %d", a.String(), resp.StatusCode)
		return nil
	}
	var ret []byte
	if ret, err = io.ReadAll(resp.Body); err != nil {
		log.Warn().Err(err).Msg("failed to read response")
		return nil
	}
	if !strings.Contains(string(ret), "Philips hue") || !strings.Contains(string(ret), "bridgeid") {
		log.Debug().Msgf("%s does not appear to be a hue bridge", a.String())
		return nil
	}

	br, _ := huego.NewCustom(ret, a.String(), http.DefaultClient)
	return br
}

func scanChoicePrompt(interfaces []net.Interface) net.Interface {
	confirmPrompt := tui.Select{
		Label:     "Choose a network interface to scan for bridges:",
		Items:     interfaces,
		CursorPos: 0,
		IsVimMode: false,
		Pointer:   common.ZiggsPointer,
	}
	choice, _, _ := confirmPrompt.Run()
	return interfaces[choice]
}

func checkAddrs(ctx context.Context, addrs []net.Addr, working *int32, resChan chan interface{}) {
	var init = &sync.Once{}
	log.Trace().Msg("checking addresses")
	for _, a := range addrs {
		log.Trace().Msgf("checking %s", a.String())
		ips := network.IterateNetRange(netaddr.MustParseIPPrefix(a.String()))
		for ipa := range ips {
			init.Do(func() { resChan <- &huego.Bridge{} })
		ctxLoop:
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if atomic.LoadInt32(working) > 25 {
						time.Sleep(100 * time.Millisecond)
						continue
					}
					break ctxLoop
				}
			}
			log.Trace().Msgf("checking %s", ipa.String())
			atomic.AddInt32(working, 1)
			go func(ip netaddr.IP) {
				resChan <- enumerateBridge(ip.IPAddr())
				time.Sleep(100 * time.Millisecond)
				atomic.AddInt32(working, -1)
			}(ipa)
		}
	}
}

// Determine the LAN network, then look for http servers on all of the local IPs.
func scanForBridges() ([]*huego.Bridge, error) {
	var hueIPs []*huego.Bridge
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	interfaces = filterCandidateInterfaces(interfaces)
	if len(interfaces) == 0 {
		return nil, errors.New("no viable interfaces found")
	}
	chosen := scanChoicePrompt(interfaces)
	var addrs []net.Addr
	if addrs, err = chosen.Addrs(); err != nil {
		log.Debug().Err(err).Msg("failed to get addresses")
		return nil, err
	}
	var working int32
	resChan := make(chan interface{}, 55)
	ctx, cancel := context.WithCancel(context.Background())
	log.Trace().Interface("addresses", addrs).Msg("checkAddrs()")
	go checkAddrs(ctx, addrs, &working, resChan)
	<-resChan // wait for sync.Once to throw us a nil

resultLoop:
	for {
		select {
		case res := <-resChan:
			bridge, ok := res.(*huego.Bridge)
			if ok && bridge != nil {
				log.Info().Msgf("found %T: %v", bridge, bridge)
				hueIPs = append(hueIPs, bridge)
				cancel()
				atomic.StoreInt32(&working, 0)
			}
		default:
			if atomic.LoadInt32(&working) <= 0 {
				cancel()
				break resultLoop
			}
		}
	}

	if len(hueIPs) == 0 {
		return nil, errors.New("no bridges found")
	}

	return hueIPs, nil
}

func promptForDiscovery() error {
	log.Warn().Msg("failed to connect to known bridges from configuration file.")
	confirmPrompt := tui.Select{
		Label:     "Search for bridges?",
		Items:     []string{"Yes", "No"},
		CursorPos: 0,
		IsVimMode: false,
		Pointer:   common.ZiggsPointer,
	}
	choice, _, _ := confirmPrompt.Run()
	if choice != 0 {
		return errNoBridges
	}
	log.Info().Msg("searching for bridges...")
	bridges, err := scanForBridges()
	if err != nil {
		return err
	}
	if len(bridges) < 1 {
		return errNoBridges
	}
	var cs []*huego.Bridge
	for _, brd := range bridges {
		cs = append(cs, brd)
	}

	Lucifer.Lock()
	defer Lucifer.Unlock()
	for _, c := range cs {
		cnt := &Bridge{
			Bridge:  c,
			RWMutex: &sync.RWMutex{},
		}
		if promptForUser(cnt) {
			log.Info().Str("caller", cnt.Host).Msg("login sucessful!")
			if err = getBridgeInfo(cnt); err != nil {
				return err
			}
		}
		Lucifer.Lock()
		Lucifer.Bridges[cnt.Info.BridgeID] = cnt
		Lucifer.Unlock()
	}
	return nil
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
