package interactive

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"time"

	"github.com/amimof/huego"
	"github.com/davecgh/go-spew/spew"

	"git.tcp.direct/kayos/ziggs/internal/system"
	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

var errInvalidFormat = errors.New("invalid format")

var (
	cpuOn     = false
	cpuCtx    context.Context
	cpuCancel context.CancelFunc
)

func ParseHexColorFast(s string) (c color.RGBA, err error) {
	c.A = 0xff

	if s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	case 4:
		c.R = hexToByte(s[1]) * 17
		c.G = hexToByte(s[2]) * 17
		c.B = hexToByte(s[3]) * 17
	default:
		err = errInvalidFormat
	}
	return
}

func cmdLights(br *ziggy.Bridge, args []string) error {
	if len(br.HueLights) == 0 {
		return errors.New("no lights found")
	}
	for _, l := range br.HueLights {
		log.Info().Str("caller", l.Name).
			Int("ID", l.ID).Str("type", l.ProductName).
			Str("model", l.ModelID).Bool("on", l.IsOn()).Msgf("%v", l.State)
	}
	return nil
}

type cmdTarget interface {
	On() error
	Off() error
	Bri(uint8) error
	Ct(uint16) error
	Hue(uint16) error
	Sat(uint8) error
	Col(color.Color) error
	SetState(huego.State) error
	Alert(string) error
}

func cmdSet(bridge *ziggy.Bridge, args []string) error {
	if len(args) < 3 {
		return errors.New("not enough arguments")
	}

	type (
		action func() error
	)

	var (
		groupmap     map[string]*huego.Group
		actions      []action
		currentState *huego.State
		argHead      = -1
		target       cmdTarget
	)

	for range args {
		argHead++
		if len(args) <= argHead {
			break
		}
		log.Trace().Int("argHead", argHead).Msg(args[argHead])
		switch args[argHead] {
		case "group", "g", "grp":
			var err error
			groupmap, err = getGroupMap()
			if err != nil {
				return err
			}
			if len(args) <= argHead-1 {
				return errors.New("no group specified")
			}
			argHead++
			g, ok := groupmap[args[argHead]]
			if !ok {
				return errors.New("group not found")
			}

			target = g
		case "on":
			actions = append(actions, target.On)
		case "off":
			actions = append(actions, target.Off)
		case "brightness--", "dim":
			actions = append(actions, func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				err := target.Bri(currentState.Bri - 5)
				if err != nil {
					err = fmt.Errorf("couldn't lower brightness: %w", err)
				}
				return err
			})
		case "brightness++", "brighten":
			actions = append(actions, func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				err := target.Bri(currentState.Bri + 5)
				if err != nil {
					err = fmt.Errorf("couldn't raise brightness: %w", err)
				}
				return err
			})
		case "brightness":
			if len(args) == argHead-1 {
				return errors.New("no brightness specified")
			}
			argHead++
			newBrightness, numErr := strconv.Atoi(args[argHead])
			if numErr != nil {
				return fmt.Errorf("given brightness is not a number: %w", numErr)
			}
			actions = append(actions, func() error {
				err := target.Bri(uint8(newBrightness))
				if err != nil {
					err = fmt.Errorf("failed to set brightness: %w", err)
				}
				return err
			})
		case "color":
			if len(args) == argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newcolor, err := ParseHexColorFast(args[argHead])
			if err != nil {
				return err
			}
			actions = append(actions, func() error {
				colErr := target.Col(newcolor)
				if colErr != nil {
					colErr = fmt.Errorf("failed to set color: %w", colErr)
				}
				return colErr
			})
		case "alert":
			actions = append(actions, func() error {
				alErr := target.Alert("select")
				if alErr != nil {
					alErr = fmt.Errorf("failed to turn on alert: %w", alErr)
				}
				return alErr
			})
		case "cpu":
			switch cpuOn {
			case false:
				cpuCtx, cpuCancel = context.WithCancel(context.Background())
				load, err := system.CPULoadGradient(cpuCtx,
					"deepskyblue", "seagreen", "darkorchid", "gold", "deeppink")
				if err != nil {
					return err
				}
				log.Info().Msg("turning CPU load lights on for ")
				go func(cpuTarget cmdTarget) {
					cpuOn = true
					defer func() {
						cpuOn = false
					}()
					for {
						select {
						case <-cpuCtx.Done():
							cpuOn = false
							return
						case clr := <-load:
							log.Trace().Msgf("CPU load color: %v", clr.Hex())
							cHex, cErr := ParseHexColorFast(clr.Hex())
							if cErr != nil {
								log.Error().Err(cErr).Msg("failed to parse color")
								continue
							}

							colErr := cpuTarget.Col(cHex)
							if colErr != nil {
								log.Error().Err(colErr).Msg("failed to set color")
								time.Sleep(3 * time.Second)
								continue
							}
						}
					}
				}(target)
				return nil
			case true:
				log.Info().Msg("turning CPU load lights off")
				cpuCancel()
				return nil
			}
		default:
			return fmt.Errorf("unknown argument: " + args[argHead])
		}
	}
	if actions == nil {
		return errors.New("no action specified")
	}
	if target == nil {
		return errors.New("no target specified")
	}
	tgroup, tgok := target.(*huego.Group)
	tlight, tlok := target.(*huego.Light)
	switch {
	case tgok:
		currentState = tgroup.State
	case tlok:
		currentState = tlight.State
	default:
		return errors.New("unknown target")
	}
	log.Trace().Msgf("current state: %v", currentState)
	for d, act := range actions {
		log.Trace().Msgf("running action %d", d)
		err := act()
		if err != nil {
			return err
		}
		switch {
		case tgok:
			currentState = tgroup.State
		case tlok:
			currentState = tlight.State
		}
		log.Trace().Msgf("new state: %v", currentState)
	}
	return nil
}

func getGroupMap() (map[string]*huego.Group, error) {
	var groupmap = make(map[string]*huego.Group)
	for _, br := range ziggy.Lucifer.Bridges {
		groups, err := br.GetGroups()
		log.Trace().Msgf(spew.Sprint(groups))
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			groupName := group.Name
			var count = 1
			for _, ok := groupmap[groupName]; ok; _, ok = groupmap[groupName] {
				groupName = fmt.Sprintf("%s_%d", group.Name, count)
			}
			groupmap[groupName] = &group
		}
	}
	return groupmap, nil
}

func getLightMap(br *ziggy.Bridge) (map[string]*huego.Light, error) {
	var lightmap = make(map[string]*huego.Light)
	ls, err := br.Bridge.GetLights()
	if err != nil {
		return nil, err
	}
	for _, l := range ls {
		lightmap[l.Name] = &l
	}
	return lightmap, nil
}

func cmdGroups(br *ziggy.Bridge, args []string) error {
	groupmap, err := getGroupMap()
	if err != nil {
		return err
	}
	if len(groupmap) == 0 {
		return errors.New("no groups found")
	}
	for n, g := range groupmap {
		if n != g.Name {
			log.Warn().Msgf("group name mismatch: %s != %s", n, g.Name)
		}
		slog := log.With().Str("caller", g.Name).Int("ID", g.ID).Logger()
		slog.Info().Msgf("\n\tType: %v\n\tClass: %v\n\t%v", g.Type, g.Class, spew.Sprint(g.State))
	}
	return nil
}
