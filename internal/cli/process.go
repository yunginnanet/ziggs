package cli

import (
	cli "git.tcp.direct/Mirrors/go-prompt"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func processGroups(grps map[string]*ziggy.HueGroup) {
	for grp, g := range grps {
		suffix := ""
		if g.Type != "" {
			suffix = " (" + g.Type + ")"
		}
		suggestionMutex.Lock()
		suggestions[2][grp] = &completion{
			Suggest: cli.Suggest{
				Text:        grp,
				Description: "Group" + suffix,
			},
			requires: map[int]map[string]bool{
				1: {"set": true, "s": true, "delete": true, "d": true, "get": true, "dump": true},
				2: {"group": true, "g": true},
			},
			root: false,
		}
		suggestionMutex.Unlock()
	}
}

func processScenes(scns map[string]*ziggy.HueScene) {
	for scn, s := range scns {
		suffix := ""
		if s.Type != "" {
			suffix = " (" + s.Type + ")"
		}
		suggestionMutex.Lock()
		suggestions[4][scn] = &completion{
			Suggest: cli.Suggest{
				Text:        scn,
				Description: "Scene" + suffix,
			},
			requires: map[int]map[string]bool{
				1: {"set": true, "s": true, "delete": true, "d": true, "get": true, "dump": true},
				2: {"group": true, "g": true, "scene": true, "s": true, "light": true, "l": true},
				4: {"scene": true, "s": true},
			},
			callback: func(args []string) bool {
				if extraDebug {
					log.Trace().Msgf("Checking if scene %s belongs to group %s, their group is %s",
						s.Name, args[3], s.Group)
				}
				if len(args) < 4 {
					return false
				}
				delGetDumpOnly := args[1] == "scene" || args[1] == "s"
				switch {
				case delGetDumpOnly && args[3] == "scene" || args[3] == "s":
					return false
				case delGetDumpOnly && args[0] == "set":
					return false
				case args[1] == "group" || args[1] == "g":
					if extraDebug {
						log.Trace().Msgf("Checking if group %s is %s", args[3], s.Group)
					}
					if args[3] == s.Group {
						return true
					}
				default:
					return false
				}
				return false
			},
			root: false,
		}
		suggestionMutex.Unlock()
	}
}

func processLights(lghts map[string]*ziggy.HueLight) {
	for lt, l := range lghts {
		suffix := ""
		if l.Type != "" {
			suffix = " (" + l.Type + ")"
		}
		suggestionMutex.Lock()
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
		suggestionMutex.Unlock()
	}
}

func processBridges() {
	for brd, b := range ziggy.Lucifer.Bridges {
		suggestionMutex.Lock()
		suggestions[1]["bridge"] = &completion{
			Suggest: cli.Suggest{
				Text:        brd,
				Description: "Bridge: " + b.Host,
			},
			requires: map[int]map[string]bool{0: {"use": true, "u": true}},
			root:     false,
		}
		suggestionMutex.Unlock()
	}
}
