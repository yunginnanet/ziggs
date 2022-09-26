package cli

import (
	"strings"

	cli "git.tcp.direct/Mirrors/go-prompt"
)

const (
	grn = "\033[32m"
	red = "\033[31m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

type completion struct {
	cli.Suggest
	inner    *ziggsCommand
	requires map[int][]string
	root     bool
}

func (c completion) qualifies(line string) bool {
	args := strings.Fields(line)
	if len(line) == 0 {
		return false
	}
	if c.root && len(args) < 1 {
		return true
	}
	/*if c.root && len(args) > 0 {
		return false
	}*/
	if len(args) < len(c.requires) {
		if extraDebug {
			log.Trace().Int("len(args)", len(args)).Int("len(c.requires)", len(c.requires)).
				Msg(red + "len(args) < len(c.requires)" + rst)
		}
		return false
	}
	if len(args)-2 > len(c.requires) {
		if extraDebug {
			log.Trace().Int("len(args)-2", len(args)-2).Int("len(c.requires)", len(c.requires)).
				Msg(red + "len(args)-2 > len(c.requires)" + rst)
		}
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
			if extraDebug {
				log.Trace().Msgf("%v%s: found %s%v", grn, c.Text, a, rst)
			}
			count++
		}
	}
	return count >= len(c.requires)
}

var suggestions map[int][]completion

func init() {
	suggestions = make(map[int][]completion)
	suggestions[0] = []completion{
		{Suggest: cli.Suggest{Text: "lights", Description: "print all known lights"}, inner: CLICommands["lights"], root: true},
		{Suggest: cli.Suggest{Text: "groups", Description: "print all known groups"}, inner: CLICommands["groups"], root: true},
		{Suggest: cli.Suggest{Text: "rules", Description: "print all known rules"}, inner: CLICommands["rules"], root: true},
		{Suggest: cli.Suggest{Text: "scenes", Description: "print all known scenes"}, inner: CLICommands["scenes"], root: true},
		{Suggest: cli.Suggest{Text: "schedules", Description: "print all known schedules"}, inner: CLICommands["schedules"], root: true},
		{Suggest: cli.Suggest{Text: "sensors", Description: "print all known sensors"}, inner: CLICommands["sensors"], root: true},
		{Suggest: cli.Suggest{Text: "set", Description: "set state of target"}, inner: CLICommands["set"], root: true},
		{Suggest: cli.Suggest{Text: "create", Description: "create object"}, inner: CLICommands["create"], root: true},
		{Suggest: cli.Suggest{Text: "delete", Description: "delete object"}, inner: CLICommands["delete"], root: true},
		{Suggest: cli.Suggest{Text: "clear", Description: "clear screen"}},
		{Suggest: cli.Suggest{Text: "scan", Description: "scan for bridges"}},
		{Suggest: cli.Suggest{Text: "exit", Description: "exit ziggs"}},
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
		sug.requires = map[int][]string{0: {"delete", "del", "set", "s"}}
		sug.root = false
	}
	delCompletion := []completion{
		{Suggest: cli.Suggest{Text: "scene", Description: "target scene"}},
		{Suggest: cli.Suggest{Text: "schedule", Description: "target schedule"}},
		{Suggest: cli.Suggest{Text: "sensor", Description: "target sensor"}},
	}
	for _, sug := range delCompletion {
		sug.requires = map[int][]string{0: {"delete", "del"}}
		sug.root = false
	}
	suggestions[1] = append(suggestions[1], delCompletion...)
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
