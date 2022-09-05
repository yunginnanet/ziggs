package interactive

import (
	"strings"

	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

type reactor func(bridge *ziggy.Bridge, args []string) error

const (
	grn = "\033[32m"
	red = "\033[31m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

var bridgeCMD = map[string]reactor{
	"scan":   cmdScan,
	"lights": cmdLights,
	"groups": cmdGroups,
	"set":    cmdSet,
}

type completion struct {
	cli.Suggest
	requires map[int][]string
	root     bool
}

func (c completion) qualifies(line string) bool {
	args := strings.Fields(line)

	if c.root && len(args) < 1 {
		return true
	}
	/*if c.root && len(args) > 0 {
		return false
	}*/
	if len(args) < len(c.requires) {
		log.Trace().Int("len(args)", len(args)).Int("len(c.requires)", len(c.requires)).
			Msg(red + "len(args) < len(c.requires)" + rst)
		return false
	}
	if len(args)-2 > len(c.requires) {
		log.Trace().Int("len(args)-2", len(args)-2).Int("len(c.requires)", len(c.requires)).
			Msg(red + "len(args)-2 > len(c.requires)" + rst)
		return false
	}
	has := func(b []string, a string) bool {
		for _, r := range b {
			if a == r {
				return true
			}
		}
		return false
	}
	var count = 0
	for i, a := range args {
		if has(c.requires[i], a) {
			log.Trace().Msgf("%v%s: found %s%v", grn, c.Text, a, rst)
			count++
		}
	}
	if count == len(c.requires) {
		return true
	}
	return false
}

var suggestions map[int][]completion

func init() {
	suggestions = make(map[int][]completion)
	suggestions[0] = []completion{
		{Suggest: cli.Suggest{Text: "lights", Description: "print all known lights"}},
		{Suggest: cli.Suggest{Text: "groups", Description: "print all known groups"}},
		{Suggest: cli.Suggest{Text: "clear", Description: "clear screen"}},
		{Suggest: cli.Suggest{Text: "scan", Description: "scan for bridges"}},
		{Suggest: cli.Suggest{Text: "exit", Description: "exit ziggs"}},
		{Suggest: cli.Suggest{Text: "quit", Description: "exit ziggs"}},
		{Suggest: cli.Suggest{Text: "set", Description: "set state of target"}},
		{Suggest: cli.Suggest{Text: "use", Description: "select bridge to perform actions on"}},
	}
	for _, sug := range suggestions[0] {
		sug.requires = map[int][]string{}
		sug.root = true
	}
	suggestions[1] = []completion{
		{Suggest: cli.Suggest{Text: "group", Description: "target group"}},
		{Suggest: cli.Suggest{Text: "light", Description: "target light"}},
	}
	for _, sug := range suggestions[1] {
		sug.requires = map[int][]string{0: {"set", "s"}}
		sug.root = false
	}
}

func processGroups(grps map[string]*huego.Group) {
	for grp, g := range grps {
		suffix := ""
		if g.Type != "" {
			suffix = " (" + g.Type + ")"
		}
		suggestions[2] = append(suggestions[2],
			completion{
				Suggest: cli.Suggest{
					Text:        grp,
					Description: "Group" + suffix,
				},
				requires: map[int][]string{
					0: {"set", "s"},
					1: {"group", "g"},
				},
				root: false,
			})
	}
}

func processLights() {
	for lt, l := range ziggy.Lucifer.Lights {
		suffix := ""
		if l.Type != "" {
			suffix = " (" + l.Type + ")"
		}
		suggestions[2] = append(suggestions[2],
			completion{
				Suggest: cli.Suggest{
					Text:        lt,
					Description: "Light" + suffix,
				},
				requires: map[int][]string{
					0: {"set", "s"},
					1: {"light", "l"},
				},
				root: false,
			})
	}
}

func processBridges() {
	for brd, b := range ziggy.Lucifer.Bridges {
		suggestions[1] = append(suggestions[1],
			completion{
				Suggest: cli.Suggest{
					Text:        brd,
					Description: "Bridge: " + b.Host,
				},
				requires: map[int][]string{0: {"use", "u"}},
				root:     false,
			})
	}
}

func completer(in cli.Document) []cli.Suggest {
	c := in.CurrentLine()
	infields := strings.Fields(c)
	var head = len(infields) - 1
	if len(infields) == 0 {
		head = 0
	}
	var sugs []cli.Suggest
	for _, sug := range suggestions[head] {
		if sug.qualifies(c) && strings.HasPrefix(sug.Text, in.GetWordBeforeCursor()) {
			sugs = append(sugs, sug.Suggest)
		}
	}
	return sugs
}
