package cli

import (
	"context"
	"errors"
	"fmt"
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
	groupmap, err := ziggy.GetGroupMap()
	if err != nil {
		return err
	}
	if len(groupmap) == 0 {
		return errors.New("no groups found")
	}
	for n, g := range groupmap {
		log.Info().Str("caller", strings.Split(br.Host, "://")[1]).
			Str("mapname", n).Str("type", g.Type).Int("ID", g.ID).
			Str("class", g.Class).Bool("on", g.IsOn()).Msgf("Group: %s", g.Name)
		log.Trace().Msgf("%v", spew.Sprint(g))
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
