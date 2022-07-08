package interactive

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"

	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/lights"
)

var errInvalidFormat = errors.New("invalid format")

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

func cmdLights(br *lights.Bridge, args []string) error {
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

func cmdSet(bridge *lights.Bridge, args []string) error {
	if len(args) < 3 {
		return errors.New("not enough arguments")
	}

	var target interface {
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

	var groupmap map[string]huego.Group

	var action func() error
	var currentState *huego.State

	var argHead = -1
	for range args {
		argHead++
		if len(args) <= argHead {
			break
		}
		log.Trace().Int("argHead", argHead).Msg(args[argHead])
		switch args[argHead] {
		case "group", "g", "grp":
			var err error
			groupmap, err = getGroupMap(bridge)
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
			target = &g
		case "on":
			action = target.On
		case "off":
			action = target.Off
		case "brightness--", "dim":
			action = func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				return fmt.Errorf("couldn't lower brightness: %w", target.Bri(currentState.Bri-5))
			}
		case "brightness++", "brighten":
			action = func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				return fmt.Errorf("couldn't raise brightness: %w", target.Bri(currentState.Bri+5))
			}
		case "brightness":
			if len(args) <= argHead-1 {
				return errors.New("no brightness specified")
			}
			argHead++
			newBrightness, numErr := strconv.Atoi(args[argHead])
			if numErr != nil {
				return numErr
			}
			action = func() error {
				return fmt.Errorf("failed to set brightness: %w",
					target.Bri(uint8(newBrightness)))
			}
		case "color":
			if len(args) <= argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newcolor, err := ParseHexColorFast(args[argHead])
			if err != nil {
				return err
			}
			action = func() error { return fmt.Errorf("failed to set color: %w", target.Col(newcolor)) }
		case "alert":
			action = func() error { return fmt.Errorf("failed to set alert: %w", target.Alert("select")) }
		default:
			return fmt.Errorf("unknown argument: " + args[argHead])
		}
	}
	if action == nil {
		return errors.New("no action specified")
	}
	if target == nil {
		return errors.New("no target specified")
	}
	tg, tgok := target.(*huego.Group)
	tl, tlok := target.(*huego.Light)
	switch {
	case tgok:
		currentState = tg.State
	case tlok:
		currentState = tl.State
	}
	log.Trace().Msgf("current state: %v", currentState)
	err := action()
	if err != nil {
		return err
	}
	switch {
	case tgok:
		currentState = tg.State
	case tlok:
		currentState = tl.State
	}
	log.Trace().Msgf("new state: %v", currentState)
	return nil
}

func getGroupMap(br *lights.Bridge) (map[string]huego.Group, error) {
	var groupmap = make(map[string]huego.Group)
	gs, err := br.Bridge.GetGroups()
	if err != nil {
		return nil, err
	}
	for _, g := range gs {
		groupmap[g.Name] = g
	}
	return groupmap, nil
}

func cmdGroups(br *lights.Bridge, args []string) error {
	groupmap, err := getGroupMap(br)
	if err != nil {
		return err
	}
	if len(groupmap) == 0 {
		return errors.New("no groups found")
	}
	for _, g := range groupmap {
		log.Info().Str("caller", g.Name).Str("type", g.Type).Int("ID", g.ID).
			Str("class", g.Class).Bool("on", g.IsOn()).Msgf("%v", g.GroupState)
	}
	return nil
}

type reactor func(bridge *lights.Bridge, args []string) error

var bridgeCMD = map[string]reactor{
	"scan":   cmdScan,
	"lights": cmdLights,
	"groups": cmdGroups,
	"set":    cmdSet,
}

const use = "use"

type completeMapper map[cli.Suggest][]cli.Suggest

var suggestions completeMapper = make(map[cli.Suggest][]cli.Suggest)

func processBridges(brs map[string]*lights.Bridge) {
	for brd, c := range brs {
		suggestions[cli.Suggest{Text: "use"}] = append(
			suggestions[cli.Suggest{Text: "use"}],
			cli.Suggest{
				Text:        brd,
				Description: c.Host,
			})
	}
}
