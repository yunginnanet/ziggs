package ziggy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amimof/huego"
	tui "github.com/manifoldco/promptui"
	"inet.af/netaddr"

	"git.tcp.direct/kayos/common/network"

	"git.tcp.direct/kayos/ziggs/internal/common"
)

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

func enumerateBridge(a net.Addr, ctx context.Context) interface{} {
	var err error
	if _, err = net.DialTimeout("tcp", a.String()+":80", 2*time.Second); err != nil {
		select {
		case <-ctx.Done():
			//
		default:
			log.Debug().Err(err).Msgf("failed to dial %s", a.String())
		}
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
	var ifaceMap = make(map[string]int)
	var ifaces []string
	for index, iface := range interfaces {
		ifaceMap[iface.Name] = index
		ifaces = append(ifaces, iface.Name)
	}
	confirmPrompt := tui.Select{
		Label:     "Choose a network interface to scan for bridges:",
		Items:     ifaces,
		CursorPos: 0,
		IsVimMode: false,
		Pointer:   common.ZiggsPointer,
	}
	_, choice, _ := confirmPrompt.Run()
	return interfaces[ifaceMap[choice]]
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
				resChan <- enumerateBridge(ip.IPAddr(), ctx)
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
		case <-ctx.Done():
			cancel()
			break resultLoop
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
		Lucifer.Bridges[cnt.Info.IPAddress] = cnt
		Lucifer.Unlock()
	}
	return nil
}
