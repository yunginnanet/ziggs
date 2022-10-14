package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	reactor     reactor
	description string
	aliases     []string
	isAlias     bool
}

type reactor func(bridge *ziggy.Bridge, args []string) error

func newZiggsCommand(react reactor, desc string, aliases ...string) *ziggsCommand {
	ret := &ziggsCommand{
		reactor:     react,
		aliases:     aliases,
		isAlias:     false,
		description: desc,
	}
	for _, alias := range aliases {
		Commands[alias] = &ziggsCommand{
			reactor: react,
			isAlias: true,
		}
	}
	return ret
}

var Commands = make(map[string]*ziggsCommand)

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
		log.Info().Str("caller", strings.Split(br.Host, "://")[1]).
			Str("ID", scene.ID).Msgf("Scene: %s", scene.Name)
		log.Trace().Msgf("%v", spew.Sprint(scene))
	}
	return nil
}

func cmdLights(br *ziggy.Bridge, args []string) error {
	for name, l := range ziggy.GetLightMap() {
		log.Info().
			Str("caller", strings.Split(br.Host, "://")[1]).Int("ID", l.ID).Str("type", l.ProductName).
			Str("model", l.ModelID).Bool("on", l.IsOn()).Msgf("Light: %s", name)
		log.Trace().Msgf("%v", spew.Sprint(l))
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
		log.Info().Str("caller", strings.Split(br.Host, "://")[1]).Int("ID", r.ID).
			Str("status", r.Status).Msgf("Rule: %s", r.Name)
		log.Trace().Msgf("%v", spew.Sprint(r))
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
		log.Info().Str("caller", strings.Split(br.Host, "://")[1]).Int("ID", s.ID).
			Str("desc", s.Description).Msgf("Schedule: %s", s.Name)
		log.Trace().Msgf("%v", spew.Sprint(s))
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
		log.Info().Str("caller", strings.Split(br.Host, "://")[1]).Int("ID", s.ID).
			Str("type", s.Type).Msgf("Sensor: %s", s.Name)
		log.Trace().Msgf("%v", spew.Sprint(s))
	}
	return nil
}

func cmdGroups(br *ziggy.Bridge, args []string) error {
	groupmap := ziggy.GetGroupMap()
	if len(groupmap) == 0 {
		return errors.New("no groups found")
	}
	for n, g := range groupmap {
		log.Info().Str("caller", g.Name).
			Str("mapname", n).Str("type", g.Type).Int("ID", g.ID).
			Str("class", g.Class).Bool("on", g.IsOn()).Send()
		for _, l := range g.Lights {
			lid, _ := strconv.Atoi(l)
			lght, err := br.GetLight(lid)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to get light %s", l)
				continue
			}
			log.Info().Msg("\t[" + strconv.Itoa(lght.ID) + "] " + lght.Name + " (" + lght.ProductName + ")")
		}
		log.Trace().Msgf("%v", spew.Sprint(g))
	}
	return nil
}

func cmdDelete(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	confirm := func() bool {
		log.Info().Msgf("Are you sure you want to delete the %s identified as %s? [y/N]", args[0], args[1])
		var input string
		fmt.Scanln(&input)
		return strings.ToLower(input) == "y"
	}
	switch args[0] {
	case "light", "l":
		t, err := br.FindLight(args[1])
		if err != nil {
			return err
		}
		if confirm() {
			return br.DeleteLight(t.ID)
		}
	case "group", "g":
		t, err := br.FindGroup(args[1])
		if err != nil {
			return err
		}
		if confirm() {
			return br.DeleteGroup(t.ID)
		}
	case "schedule":
		if confirm() {
			if argID, err := strconv.Atoi(args[1]); err == nil {
				return br.DeleteSchedule(argID)
			} else {
				return err
			}
		}
	case "rule":
		if argID, err := strconv.Atoi(args[1]); err == nil {
			return br.DeleteSchedule(argID)
		} else {
			return err
		}
	case "sensor":
		if argID, err := strconv.Atoi(args[1]); err == nil {
			return br.DeleteSchedule(argID)
		} else {
			return err
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
	type renameable interface {
		Rename(string) error
	}
	var (
		target renameable
		err    error
	)
	switch args[0] {
	case "light", "l":
		target, err = br.FindLight(args[1])
	case "group", "g":
		target, err = br.FindGroup(args[1])
	case "schedule":
		return errors.New("not implemented")
	case "rule":
		return errors.New("not implemented")
	case "sensor":
		return errors.New("not implemented")
	default:
		return errors.New("invalid target type")
	}
	if err != nil {
		return err
	}
	return target.Rename(args[2])
}

// cmdDump exports a target object to a JSON file
func cmdDump(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 && args[0] != "all" {
		return errors.New("not enough arguments")
	}
	var (
		target interface{}
		name   string
		err    error
	)
	switch args[0] {
	case "light", "l":
		target, err = br.FindLight(args[1])
		if err != nil {
			return err
		}
		name = target.(*huego.Light).Name
	case "group", "g":
		target, err = br.FindGroup(args[1])
		if err != nil {
			return err
		}
		name = target.(*huego.Group).Name
	case "schedule":
		return errors.New("not implemented")
	case "rule":
		return errors.New("not implemented")
	case "sensor":
		return errors.New("not implemented")
	case "bridge", "all":
		target = br
		name = br.Info.Name
	default:
		return errors.New("invalid target type")
	}

	if js, err := json.Marshal(target); err != nil {
		return err
	} else {
		return os.WriteFile(name, js, 0o666)
	}

}

// cmdLoad imports a target JSON object and attempts to apply it to an existing object
func cmdLoad(br *ziggy.Bridge, args []string) error {
	if len(args) < 1 {
		return errors.New("not enough arguments")
	}
	js, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	switch args[0] {
	case "light", "l":
		target, err := br.FindLight(args[1])
		if err != nil {
			return err
		}
		var l *huego.Light
		if err := json.Unmarshal(js, &l); err != nil {
			return err
		}
		if resp, err := br.UpdateLight(target.ID, *l); err != nil {
			return err
		} else {
			log.Info().Msgf("%v", resp)
		}
	case "group", "g":
		target, err := br.FindGroup(args[1])
		if err != nil {
			return err
		}
		var g *huego.Group
		if err := json.Unmarshal(js, &g); err != nil {
			return err
		}
		if resp, err := br.UpdateGroup(target.ID, *g); err != nil {
			return err
		} else {
			log.Info().Msgf("%v", resp)
		}
	case "schedule":
		return errors.New("not implemented")
	case "rule":
		return errors.New("not implemented")
	case "sensor":
		return errors.New("not implemented")
	case "bridge":
		return errors.New("not implemented")
	default:
		return errors.New("invalid target type")
	}
	return nil
}

func cmdAdopt(br *ziggy.Bridge, args []string) error {
	resp, err := br.FindLights()
	if err != nil {
		return err
	}
	log.Debug().Msgf(spew.Sprint(resp.Success))
	newLights, err := br.GetNewLights()
	if err != nil {
		return err
	}
	print("searching")
	for count := 0; count < 10; count++ {
		print(".")
		time.Sleep(1 * time.Second)
	}
	if len(newLights.Lights) == 0 {
		return errors.New("no new lights found")
	}
	for _, l := range newLights.Lights {
		log.Info().Msgf("[+] %s", l)
		log.Trace().Msgf("%v", spew.Sprint(l))
	}
	return nil
}
