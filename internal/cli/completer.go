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
	requires map[int][]string
	isAlias  bool
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

var suggestions map[int][]*completion

func init() {
	Commands["ls"] = newZiggsCommand(cmdList, "list all lights, groups, scenes, rules, and schedules")
	Commands["schedules"] = newZiggsCommand(cmdSchedules, "list schedules", "lssched", "crontab")
	Commands["rules"] = newZiggsCommand(cmdRules, "list rules", "lsrule")
	Commands["sensors"] = newZiggsCommand(cmdSensors, "list sensors", "lssens")
	Commands["scenes"] = newZiggsCommand(cmdScenes, "list scenes", "lsscene")
	Commands["lights"] = newZiggsCommand(cmdLights, "list lights", "lslight")
	Commands["groups"] = newZiggsCommand(cmdGroups, "list groups", "lsgrp")
	Commands["create"] = newZiggsCommand(cmdCreate, "create a new object in bridge", "new", "mk")
	Commands["delete"] = newZiggsCommand(cmdDelete, "delete objects from bridges", "del", "remove")
	Commands["scan"] = newZiggsCommand(cmdScan, "scan for bridges/lights/sensors", "search", "find")
	Commands["rename"] = newZiggsCommand(cmdRename, "rename object in bridge", "mv")
	Commands["adopt"] = newZiggsCommand(cmdAdopt, "adopt new lights to the bridge")
	Commands["dump"] = newZiggsCommand(cmdDump, "dump target object JSON to a file")
	Commands["load"] = newZiggsCommand(cmdLoad, "load JSON from a file into the bridge")
	Commands["set"] = newZiggsCommand(cmdSet, "update object properties in bridge")
	Commands["fwupdate"] = newZiggsCommand(cmdFirmwareUpdate, "inform bridge to check for updates",
		"fwup", "upgrade")
	Commands["info"] = newZiggsCommand(cmdInfo, "show information about a bridge", "uname")
	initCompletion()
}

func initCompletion() {
	suggestions = make(map[int][]*completion)
	suggestions[0] = []*completion{
		{Suggest: cli.Suggest{Text: "lights"}, inner: Commands["lights"]},
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
		{Suggest: cli.Suggest{Text: "clear", Description: "clear screen"}},
		{Suggest: cli.Suggest{Text: "exit", Description: "exit ziggs"}},
	}

	for _, sug := range suggestions[0] {
		sug.requires = map[int][]string{}
		sug.root = true
		if sug.inner != nil {
			sug.Suggest.Description = sug.inner.description
		}
		if sug.inner != nil && len(sug.inner.aliases) > 0 {
			for _, a := range sug.inner.aliases {
				suggestions[0] = append(suggestions[0], &completion{
					Suggest: cli.Suggest{Text: a, Description: sug.Description},
					inner:   sug.inner,
					root:    true,
					isAlias: true,
				})
			}
		}
	}

	suggestions[1] = []*completion{
		{Suggest: cli.Suggest{Text: "group", Description: "target group"}},
		{Suggest: cli.Suggest{Text: "light", Description: "target light"}},
		{Suggest: cli.Suggest{Text: "scene", Description: "target scene"}},
		{Suggest: cli.Suggest{Text: "schedule", Description: "target schedule"}},
		{Suggest: cli.Suggest{Text: "sensor", Description: "target sensor"}},
		{Suggest: cli.Suggest{Text: "config", Description: "target bridge config"}},
	}
	for _, sug := range suggestions[1] {
		sug.requires = map[int][]string{0: {"delete", "del", "set", "s", "rename", "mv", "dump", "load"}}
		sug.root = false
	}
	delCompletion := []*completion{
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
