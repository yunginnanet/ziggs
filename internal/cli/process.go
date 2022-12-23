package cli

import (
	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func processGroups(grps map[string]*huego.Group) {
	for grp, g := range grps {
		suffix := ""
		if g.Type != "" {
			suffix = " (" + g.Type + ")"
		}

		suggestions[2][grp] = &completion{
			Suggest: cli.Suggest{
				Text:        grp,
				Description: "Group" + suffix,
			},
			requires: map[int]map[string]bool{
				1: {"set": true, "s": true, "delete": true, "d": true},
				2: {"group": true, "g": true},
			},
			root: false,
		}
	}
}

func processLights() {
	for lt, l := range ziggy.GetLightMap() {
		suffix := ""
		if l.Type != "" {
			suffix = " (" + l.Type + ")"
		}
		suggestions[2][lt] = &completion{
			Suggest: cli.Suggest{
				Text:        lt,
				Description: "Light" + suffix,
			},
			requires: map[int]map[string]bool{
				1: {"set": true, "s": true, "delete": true, "d": true},
				2: {"light": true, "l": true},
			},
			root: false,
		}
	}
}

func processBridges() {
	for brd, b := range ziggy.Lucifer.Bridges {
		suggestions[1]["bridge"] = &completion{
			Suggest: cli.Suggest{
				Text:        brd,
				Description: "Bridge: " + b.Host,
			},
			requires: map[int]map[string]bool{0: {"use": true, "u": true}},
			root:     false,
		}
	}
}
