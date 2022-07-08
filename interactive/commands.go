package interactive

import (
	"errors"
	"strconv"

	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/lights"
)

func cmdLights(br *lights.Bridge, args []string) error {
	if len(br.HueLights) <= 0 {
		return errors.New("no lights found")
	}
	for _, l := range br.HueLights {
		log.Info().Str("caller", l.Name).
			Int("ID", l.ID).Str("type", l.ProductName).
			Str("model", l.ModelID).Bool("on", l.IsOn()).Msgf("%v", l.State)
	}
	return nil
}

func cmdGroups(br *lights.Bridge, args []string) error {
	var groupmap = make(map[string]*huego.Group)
	gs, err := br.Bridge.GetGroups()
	if err != nil {
		return err
	}
	for _, g := range gs {
		groupmap[g.Name] = &g
	}

	if len(args) > 0 {
		switch {
		case args[1] == "+":
			return groupmap[args[0]].Bri(groupmap[args[0]].State.Bri + 25)
		case args[1] == "-":
			return groupmap[args[0]].Bri(groupmap[args[0]].State.Bri - 25)
		default:
			newBrightness, numErr := strconv.Atoi(args[1])
			if numErr != nil {
				return numErr
			}
			return groupmap[args[0]].Bri(uint8(newBrightness))
		}
	}

	if len(gs) == 0 {
		return errors.New("no lights found")
	}
	for _, g := range gs {
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
