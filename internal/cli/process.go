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
		suggestions[2] = append(suggestions[2],
			&completion{
				Suggest: cli.Suggest{
					Text:        grp,
					Description: "Group" + suffix,
				},
				requires: map[int][]string{
					0: {"set", "s", "delete", "d"},
					1: {"group", "g"},
				},
				root: false,
			})
	}
}

func processLights() {
	for lt, l := range ziggy.GetLightMap() {
		suffix := ""
		if l.Type != "" {
			suffix = " (" + l.Type + ")"
		}
		suggestions[2] = append(suggestions[2],
			&completion{
				Suggest: cli.Suggest{
					Text:        lt,
					Description: "Light" + suffix,
				},
				requires: map[int][]string{
					0: {"set", "s", "delete", "d", "rename", "r"},
					1: {"light", "l"},
				},
				root: false,
			})
	}
}

func processBridges() {
	for brd, b := range ziggy.Lucifer.Bridges {
		suggestions[1] = append(suggestions[1],
			&completion{
				Suggest: cli.Suggest{
					Text:        brd,
					Description: "Bridge: " + b.Host,
				},
				requires: map[int][]string{0: {"use", "u"}},
				root:     false,
			})
	}
}
