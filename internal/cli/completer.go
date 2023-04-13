package cli

import (
	"strings"
	"sync"

	cli "git.tcp.direct/Mirrors/go-prompt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/shlex"
)

const (
	grn = "\033[32m"
	red = "\033[31m"
	rst = "\033[0m"
)

type completion struct {
	cli.Suggest
	inner    *ziggsCommand
	requires map[int]map[string]bool
	callback func([]string) bool
	isAlias  bool
	root     bool
}

func (c completion) qualifies(line string) bool {
	args, err := shlex.Split(line)
	if err != nil {
		return false
	}

	verbose := func(msg string, args ...interface{}) {
		if !extraDebug {
			return
		}
		log.Trace().Caller(1).
			Int("len(args)", len(args)).
			Int("len(c.requires)", len(c.requires)).Msgf(msg, args...)
	}

	if extraDebug {
		spew.Dump(args)
	}

	switch {
	case len(args) <= 1 && c.root:
		verbose("%v%s: len(args) <= 1 && c.root", grn, c.Text)
		return true
	case len(args) < len(c.requires):
		verbose(red + "len(args) < len(c.requires)" + rst)
		return false
	case len(args)-2 > len(c.requires):
		verbose(red + "len(args)-2 > len(c.requires)" + rst)
		return false
	default:
		//
	}

	var count = 0
	for i, a := range args {
		i++
		if _, ok := c.requires[i][a]; ok {
			verbose("%v%s: found %s (count++) %v", grn, c.Text, a, rst)
			count++
		}
	}

	ok := count >= len(c.requires)
	if !ok {
		verbose("%v%s: count(%d) < len(c.requires)(%d)", red, c.Text, count, len(c.requires))
		return false
	}

	if c.callback == nil {
		return true
	}

	return c.callback(args)
}

var (
	suggestions     map[int]map[string]*completion
	suggestionMutex = &sync.RWMutex{}
)

func init() {
	Commands["ls"] = newZiggsCommand(cmdList, "list all lights, groups, scenes, rules, and schedules", 0)
	Commands["schedules"] = newZiggsCommand(cmdSchedules, "list schedules", 0,
		"lssched", "crontab")
	Commands["rules"] = newZiggsCommand(cmdRules, "list rules", 0, "lsrule")
	Commands["sensors"] = newZiggsCommand(cmdSensors, "list sensors", 0, "lssens")
	Commands["scenes"] = newZiggsCommand(cmdScenes, "list scenes", 0, "lsscene")
	Commands["lights"] = newZiggsCommand(cmdLights, "list lights", 0, "lslight")
	Commands["groups"] = newZiggsCommand(cmdGroups, "list groups", 0, "lsgrp")
	Commands["create"] = newZiggsCommand(cmdCreate, "create a new object in bridge", 3,
		"new", "mk")
	Commands["delete"] = newZiggsCommand(cmdDelete, "delete objects from bridges", 2,
		"del", "remove", "rm")
	Commands["scan"] = newZiggsCommand(cmdScan, "scan for bridges/lights/sensors", 0,
		"search", "find")
	Commands["rename"] = newZiggsCommand(cmdRename, "rename object in bridge", 3, "mv")
	Commands["cp"] = newZiggsCommand(cmdCp, "copy object pointer to a new group", 2, "copy")
	Commands["adopt"] = newZiggsCommand(cmdAdopt, "adopt new lights to the bridge", 0)
	Commands["dump"] = newZiggsCommand(cmdDump, "dump target object JSON to a file", 1)
	Commands["load"] = newZiggsCommand(cmdLoad, "load JSON from a file into the bridge", 1)
	Commands["set"] = newZiggsCommand(cmdSet, "update object properties in bridge", 3)
	Commands["get"] = newZiggsCommand(cmdGet, "get object properties from bridge", 2)
	Commands["upgrade"] = newZiggsCommand(cmdFirmwareUpdate, "inform bridge to check for updates", 0,
		"fwup", "upgrade", "fwupdate")
	Commands["info"] = newZiggsCommand(cmdInfo, "show information about a bridge", 0, "uname")
	initCompletion()
	Commands["reboot"] = newZiggsCommand(cmdReboot, "reboot bridge", 0)
}

func initCompletion() {
	suggestionMutex.Lock()
	defer suggestionMutex.Unlock()

	suggestions = make(map[int]map[string]*completion)
	suggestions[0] = make(map[string]*completion)
	suggestions[1] = make(map[string]*completion)
	suggestions[2] = make(map[string]*completion)
	suggestions[3] = make(map[string]*completion)
	suggestions[4] = make(map[string]*completion)

	/*	{Suggest: cli.Suggest{Text: "lights"}, inner: Commands["lights"]},
		{Suggest: cli.Suggest{Text: "groups"}, inner: Commands["groups"]},
		{Suggest: cli.Suggest{Text: "rules"}, inner: Commands["rules"]},
		{Suggest: cli.Suggest{Text: "scenes"}, inner: Commands["scenes"]},
		{Suggest: cli.Suggest{Text: "schedules"}, inner: Commands["schedules"]},
		{Suggest: cli.Suggest{Text: "sensors"}, inner: Commands["sensors"]},
		{Suggest: cli.Suggest{Text: "set"}, inner: Commands["set"]},
		{Suggest: cli.Suggest{Text: "create"}, inner: Commands["create"]},
		{Suggest: cli.Suggest{Text: "delete"}, inner: Commands["delete"]},
		{Suggest: cli.Suggest{Text: "scan"}, inner: Commands["scan"]},
		{Suggest: cli.Suggest{Text: "rename"}, inner: Commands["rename"]},
		{Suggest: cli.Suggest{Text: "adopt"}, inner: Commands["adopt"]},
		{Suggest: cli.Suggest{Text: "dump"}, inner: Commands["dump"]},
		{Suggest: cli.Suggest{Text: "load"}, inner: Commands["load"]},
		{Suggest: cli.Suggest{Text: "use", Description: "select bridge to perform actions on"}},
		{Suggest: cli.Suggest{Text: "exit", Description: "exit ziggs"}},
	*/

	suggestions[0]["clear"] = &completion{Suggest: cli.Suggest{Text: "clear", Description: "clear screen"}}
	suggestions[0]["exit"] = &completion{Suggest: cli.Suggest{Text: "exit", Description: "exit ziggs"}}
	suggestions[0]["help"] = &completion{Suggest: cli.Suggest{Text: "help", Description: "show help"}}

	for name, cmd := range Commands {
		suggestions[0][name] = &completion{Suggest: cli.Suggest{Text: name}, inner: cmd, root: cmd.requires == 0}
		if cmd.description != "" {
			suggestions[0][name].Description = cmd.description
		}
		suggestions[0][name].requires = map[int]map[string]bool{}
	}

	suggestions[1] = map[string]*completion{
		"group":    {Suggest: cli.Suggest{Text: "group", Description: "target group"}},
		"light":    {Suggest: cli.Suggest{Text: "light", Description: "target light"}},
		"scene":    {Suggest: cli.Suggest{Text: "scene", Description: "target scene"}},
		"schedule": {Suggest: cli.Suggest{Text: "schedule", Description: "target schedule"}},
		"sensor":   {Suggest: cli.Suggest{Text: "sensor", Description: "target sensor"}},
		"config":   {Suggest: cli.Suggest{Text: "config", Description: "target bridge config"}},
	}
	for _, sug := range suggestions[1] {
		sug.requires = map[int]map[string]bool{1: {
			"delete": true, "del": true, "set": true, "s": true,
			"rename": true, "mv": true, "dump": true, "load": true,
			"get": true,
		}}
		sug.root = false
	}
	delCompletion := []*completion{
		{Suggest: cli.Suggest{Text: "scene", Description: "target scene"}},
		{Suggest: cli.Suggest{Text: "schedule", Description: "target schedule"}},
		{Suggest: cli.Suggest{Text: "sensor", Description: "target sensor"}},
		{Suggest: cli.Suggest{Text: "group", Description: "target group"}},
	}
	for _, sug := range delCompletion {
		sug.requires = map[int]map[string]bool{1: {"delete": true, "del": true}}
		sug.root = false
	}
}

func completer(in cli.Document) []cli.Suggest {
	c := in.Text

	infields, _ := shlex.Split(c)
	var head = len(infields) - 1
	if head < 0 {
		head = 0
	}
	if head > 0 && in.LastKeyStroke() == ' ' {
		head++
	}

	if extraDebug {
		log.Trace().Int("head", head).Msgf("completing %v", infields)
	}
	var sugs []cli.Suggest
	suggestionMutex.RLock()
	defer suggestionMutex.RUnlock()
	for _, sug := range suggestions[head] {
		if !sug.qualifies(c) {
			continue
		}
		if in.TextBeforeCursor() != "" && strings.Contains(strings.ToLower(sug.Text),
			strings.ToLower(strings.TrimSpace(in.GetWordBeforeCursorWithSpace()))) {
			sugs = append(sugs, sug.Suggest)
		}
	}
	return sugs
}
