package interactive

import (
	cli "git.tcp.direct/Mirrors/go-prompt"

	"git.tcp.direct/kayos/ziggs/lights"
)

func cmdLights(br *lights.Bridge, args []string) error {
	return nil
}

type reactor func(bridge *lights.Bridge, args []string) error

var bridgeCMD = map[string]reactor{
	"scan":   cmdScan,
	"lights": cmdLights,
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
