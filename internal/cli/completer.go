package cli

import (
	"strings"

	cli "git.tcp.direct/Mirrors/go-prompt"
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
	isAlias  bool
	root     bool
}

func (c completion) qualifies(line string) bool {
	args := strings.Fields(line)

	if len(args) <= 1 && c.root {
		return true
	}

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

	var count = 0
	for i, a := range args {
		i++
		if _, ok := c.requires[i][a]; ok {
			if extraDebug {
				log.Trace().Msgf("%v%s: found %s%v", grn, c.Text, a, rst)
			}
			count++
		}
	}

	if extraDebug && !(count >= len(c.requires)) {
		log.Trace().Msgf("%v%s: count(%d) < len(c.requires)(%d)", red, c.Text, count, len(c.requires))
	}

	return count >= len(c.requires)
}

var suggestions map[int]map[string]*completion

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
	suggestions = make(map[int]map[string]*completion)
	suggestions[0] = make(map[string]*completion)
	suggestions[1] = make(map[string]*completion)
	suggestions[2] = make(map[string]*completion)

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
	c := in.CurrentLine()

	infields := strings.Fields(c)
	var head = len(infields) - 1
	if head < 0 {
		head = 0
	}
	if head == 1 {
		head = 1
	}
	if head > 0 && in.LastKeyStroke() == ' ' {
		head++
	}

	if extraDebug {
		log.Trace().Int("head", head).Msgf("completing %v", infields)
	}
	var sugs []cli.Suggest
	for _, sug := range suggestions[head] {
		if sug.qualifies(c) {
			if in.GetWordBeforeCursor() != "" && strings.HasPrefix(sug.Text, in.GetWordBeforeCursor()) {
				sugs = append(sugs, sug.Suggest)
			}
		}
	}
	return sugs
}
