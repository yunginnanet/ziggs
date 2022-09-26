package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/amimof/huego"
	"github.com/davecgh/go-spew/spew"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

var (
	cpuOn      = false
	cpuCtx     context.Context
	cpuCancel  context.CancelFunc
	cpuLastCol string
	cpuLastHue = make(map[int]uint16)
)

type ziggsCommand struct {
	reactor reactor
	aliases []string
	isAlias bool
}

type reactor func(bridge *ziggy.Bridge, args []string) error

func newZiggsCommand(
	react reactor,
	aliases ...string) *ziggsCommand {
	ret := &ziggsCommand{
		reactor: react,
		isAlias: false,
	}
	for _, alias := range aliases {
		CLICommands[alias] = &ziggsCommand{
			reactor: react,
			isAlias: true,
		}
	}
	return ret
}

var CLICommands = make(map[string]*ziggsCommand)

func init() {
	CLICommands["ls"] = newZiggsCommand(cmdList)
	CLICommands["schedules"] = newZiggsCommand(cmdSchedules, "lssched", "crontab")
	CLICommands["rules"] = newZiggsCommand(cmdRules, "lsrule")
	CLICommands["sensors"] = newZiggsCommand(cmdSensors, "lssens")
	CLICommands["scenes"] = newZiggsCommand(cmdScenes, "lsscene")
	CLICommands["lights"] = newZiggsCommand(cmdLights, "lslight")
	CLICommands["groups"] = newZiggsCommand(cmdGroups, "lsgrp")
	CLICommands["delete"] = newZiggsCommand(cmdDelete, "del", "remove")
	CLICommands["scan"] = newZiggsCommand(cmdScan, "search", "find")
	CLICommands["rename"] = newZiggsCommand(cmdRename, "mv")
	CLICommands["set"] = newZiggsCommand(cmdSet, "update")
}

func cmdList(br *ziggy.Bridge, args []string) error {
	var runs = []reactor{cmdLights, cmdGroups, cmdScenes, cmdSensors}
	var cont = false
	for _, arg := range args {
		if len(arg) > 4 {
			continue
		}
		if !strings.ContainsAny(arg, "-la") {
			continue
		}
		cont = true
		break
	}
	if cont {
		runs = append(runs, cmdSchedules, cmdRules)
	}
	for _, run := range runs {
		if err := run(br, args); err != nil {
			return err
		}
	}
	return nil
}

func cmdScenes(br *ziggy.Bridge, args []string) error {
	scenes, err := br.GetScenes()
	if err != nil {
		return err
	}
	for _, scene := range scenes {
		log.Info().Str("caller", scene.Name).Msgf("%v", spew.Sprint(scene))
	}
	return nil
}

func cmdLights(br *ziggy.Bridge, args []string) error {
	for name, l := range ziggy.GetLightMap() {
		log.Info().
			Int("ID", l.ID).Str("type", l.ProductName).
			Str("model", l.ModelID).Bool("on", l.IsOn()).Msgf("[+] %s", name)
	}
	return nil
}

func cmdRules(br *ziggy.Bridge, args []string) error {
	rules, err := br.GetRules()
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return errors.New("no rules found")
	}
	for _, r := range rules {
		log.Info().Str("caller", r.Name).Int("ID", r.ID).Msgf("%v", spew.Sprint(r))
	}
	return nil
}

func cmdSchedules(br *ziggy.Bridge, args []string) error {
	schedules, err := br.GetSchedules()
	if err != nil {
		return err
	}
	if len(schedules) == 0 {
		return errors.New("no schedules found")
	}
	for _, s := range schedules {
		log.Info().Str("caller", s.Name).Int("ID", s.ID).Msgf("%v", spew.Sprint(s))
	}
	return nil
}

func cmdSensors(br *ziggy.Bridge, args []string) error {
	sensors, err := br.GetSensors()
	if err != nil {
		return err
	}
	if len(sensors) == 0 {
		return errors.New("no sensors found")
	}
	for _, s := range sensors {
		log.Info().Str("caller", s.Name).Int("ID", s.ID).Msgf("%v", spew.Sprint(s))
	}
	return nil
}

func cmdGroups(br *ziggy.Bridge, args []string) error {
	groupmap, err := ziggy.GetGroupMap()
	if err != nil {
		return err
	}
	if len(groupmap) == 0 {
		return errors.New("no groups found")
	}
	for n, g := range groupmap {
		log.Info().Str("caller", g.Name).Str("mapname", n).Str("type", g.Type).Int("ID", g.ID).
			Str("class", g.Class).Bool("on", g.IsOn()).Msgf("%v", g.GroupState)
	}
	return nil
}

func cmdDelete(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	argID, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	confirm := func() bool {
		log.Info().Msgf("Are you sure you want to delete %s with ID %d? [y/N]", args[0], argID)
		var input string
		fmt.Scanln(&input)
		return strings.ToLower(input) == "y"
	}
	switch args[0] {
	case "light":
		if confirm() {
			return br.DeleteLight(argID)
		}
	case "group":
		if confirm() {
			return br.DeleteGroup(argID)
		}
	case "schedule":
		if confirm() {
			return br.DeleteSchedule(argID)
		}
	case "rule":
		if confirm() {
			return br.DeleteRule(argID)
		}
	case "sensor":
		if confirm() {
			return br.DeleteSensor(argID)
		}
	default:
		return errors.New("invalid target type")
	}
	return nil
}

func cmdRename(br *ziggy.Bridge, args []string) error {
	if len(args) < 3 {
		return errors.New("not enough arguments")
	}
	argID, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	switch args[0] {
	case "light":
		resp, err := br.UpdateLight(argID, huego.Light{Name: args[2]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "group":
		resp, err := br.UpdateGroup(argID, huego.Group{Name: args[2]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "schedule":
		resp, err := br.UpdateSchedule(argID, &huego.Schedule{Name: args[2]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "rule":
		resp, err := br.UpdateRule(argID, &huego.Rule{Name: args[2]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "sensor":
		resp, err := br.UpdateSensor(argID, &huego.Sensor{Name: args[2]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	default:
		return errors.New("invalid target type")
	}
	return nil
}
