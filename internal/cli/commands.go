package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/davecgh/go-spew/spew"
	"github.com/yunginnanet/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

var (
	cpuOn      = false
	cpuCtx     context.Context
	cpuCancel  context.CancelFunc
	cpuLastCol string
	cpuLastHue = make(map[int]uint16)
)

type reactor func(bridge *ziggy.Bridge, args []string) error

type ziggsCommand struct {
	reactor     reactor
	description string
	aliases     []string
	isAlias     bool
	requires    int // number of arguments required
}

func newZiggsCommand(react reactor, desc string, requires int, aliases ...string) *ziggsCommand {
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

func cmdRefresh(br *ziggy.Bridge, args []string) error {
	ziggy.NeedsUpdate()
	ziggy.GetGroupMap()
	ziggy.GetLightMap()
	ziggy.GetSceneMap()
	ziggy.GetSensorMap()
	return nil
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
	var targGroup *ziggy.HueGroup
	if len(args) > 0 {
		targGroup = ziggy.GetGroupMap()[args[0]]
	}
	scenes, err := br.GetScenes()
	if err != nil {
		return err
	}
	for _, scene := range scenes {
		scGrNum, numErr := strconv.Atoi(scene.Group)
		if numErr != nil {
			continue
		}
		grp, gerr := br.GetGroup(scGrNum)
		if gerr == nil {
			scene.Group = grp.Name
		}
		if gerr == nil && targGroup != nil {
			if targGroup.ID != scGrNum {
				continue
			}
		}
		/*for _, lstate := range scene.LightStates {
			// lstate.
		}*/

		log.Info().Str("caller", scene.Group).
			Str("ID", scene.ID).Msgf("Scene: %s", scene.Name)
		log.Trace().Caller().Msgf("%v", spew.Sprint(scene))
	}
	return nil
}

func cmdLights(br *ziggy.Bridge, args []string) error {
	for name, l := range ziggy.GetLightMap() {
		log.Info().
			Str("caller", strings.Split(br.Host, "://")[1]).Int("ID", l.ID).Str("type", l.ProductName).
			Str("model", l.ModelID).Bool("on", l.IsOn()).Msgf("Light: %s", name)
		log.Trace().Caller().Msgf("%v", spew.Sprint(l))
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
		log.Trace().Caller().Msgf("%v", spew.Sprint(r))
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
		log.Trace().Caller().Msgf("%v", spew.Sprint(s))
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
		log.Trace().Caller().Msgf("%v", spew.Sprint(s))
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
		log.Trace().Caller().Msgf("%v", spew.Sprint(g))
	}
	return nil
}

func cmdDelete(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}

	defer ziggy.NeedsUpdate()

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

// cp <light> <group>
// dump the json for a group, then create a new group that adopts only the lights, then add our light id to the new group
// then use update to push the new group to the bridge
func cmdCp(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}

	defer ziggy.NeedsUpdate()

	var (
		targetLight *ziggy.HueLight
		targetGroup *ziggy.HueGroup
		err         error
	)
	if targetLight, err = br.FindLight(args[0]); err != nil {
		return err
	}
	if targetGroup, err = br.FindGroup(args[1]); err != nil {
		return err
	}
	// dump the group
	var resp *huego.Response
	if resp, err = br.UpdateGroup(targetGroup.ID, huego.Group{
		Name:   targetGroup.Name,
		ID:     targetGroup.ID,
		Lights: append(targetGroup.Lights, strconv.Itoa(targetLight.ID)),
	}); err != nil {
		return err
	}
	log.Info().Msgf("updated group %s to include light %s", targetGroup.Name, targetLight.Name)
	log.Trace().Caller().Msgf("%v", spew.Sprint(resp))
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

	defer ziggy.NeedsUpdate()

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
		target, err = br.FindSensor(args[1])
	default:
		return errors.New("invalid target type")
	}
	if err != nil {
		return err
	}
	return target.Rename(args[2])
}

type dumpTarget struct {
	Name         string
	Object       any
	ParentFolder string
}

func newTarget(name string, obj any) dumpTarget {
	return dumpTarget{
		Name:   name,
		Object: obj,
	}
}

// cmdDump exports a target object to a JSON file
func cmdDump(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 && args[0] != "all" && args[0] != "conf" && args[0] != "groups" &&
		args[0] != "lights" && args[0] != "rules" && args[0] != "schedules" &&
		args[0] != "sensors" && args[0] != "scenes" && args[0] != "resourcelinks" &&
		args[0] != "config" {
		return errors.New("not enough arguments")
	}
	var (
		targets []dumpTarget
		err     error
	)
	switch args[0] {
	case "light", "l":
		var name string
		var target any
		target, err = br.FindLight(args[1])
		if err != nil {
			return err
		}
		lght, ok := target.(*huego.Light)
		if !ok {
			name = target.(*ziggy.HueLight).Name
		} else {
			name = lght.Name
		}
		targets = append(targets, newTarget(name, target))
	case "group", "g":
		var target any
		target, err = br.GetGroups()
		if err != nil {
			return err
		}
		for _, g := range target.([]huego.Group) {
			if !strings.EqualFold(g.Name, args[1]) && !strings.EqualFold(strconv.Itoa(g.ID), args[1]) {
				continue
			}
			targets = append(targets, newTarget(g.Name, g))
		}
	case "groups":
		var target any
		target, err = br.GetGroups()
		if err != nil {
			return err
		}
		for _, g := range target.([]huego.Group) {
			targets = append(targets, newTarget(g.Name, g))
		}
	case "schedule":
		var target any
		target, err = br.GetSchedules()
		if err != nil {
			return err
		}
		for _, s := range target.([]huego.Schedule) {
			if !strings.EqualFold(s.Name, args[1]) && !strings.EqualFold(strconv.Itoa(s.ID), args[1]) {
				continue
			}
			name := s.Name
			targets = append(targets, newTarget(name, s))
			break
		}
	case "rule":
		var target any
		target, err = br.GetRules()
		if err != nil {
			return err
		}
		for _, r := range target.([]huego.Rule) {
			if !strings.EqualFold(r.Name, args[1]) && !strings.EqualFold(strconv.Itoa(r.ID), args[1]) {
				continue
			}
			name := r.Name
			targets = append(targets, newTarget(name, r))
			break
		}
	case "rules":
		var target any
		target, err = br.GetRules()
		if err != nil {
			return err
		}
		name := "rules"
		targets = append(targets, newTarget(name, target))
	case "scenes":
		var scenes []huego.Scene
		scenes, err = br.GetScenes()
		if err != nil {
			return err
		}
		for _, s := range scenes {
			var scene *huego.Scene
			if scene, err = br.GetScene(s.ID); err != nil {
				return err
			}
			var group *huego.Group
			var num int
			if num, err = strconv.Atoi(scene.Group); err != nil {
				group = nil
			}
			group, err = br.GetGroup(num)
			if err != nil {
				group = nil
			}

			sc := newTarget(scene.Name, scene)
			if group != nil {
				sc.ParentFolder = group.Name
			}

			if mergo.Merge(&sc.Object, s); err != nil {
				return err
			}
			targets = append(targets, sc)
		}
	case "schedules":
		var target any
		target, err = br.GetSchedules()
		if err != nil {
			return err
		}
		for _, s := range target.([]huego.Schedule) {
			name := s.Name
			targets = append(targets, newTarget(name, s))
		}
	case "sensor":
		var target any
		target, err = br.GetSensors()
		if err != nil {
			return err
		}
		for _, sensor := range target.([]huego.Sensor) {
			name := sensor.Name
			targets = append(targets, newTarget(name, sensor))
		}
	case "bridge", "all":
		targets = append(targets, newTarget("bridge", br))
	case "config":
		var conf *huego.Config
		conf, err = br.GetConfig()
		if err != nil {
			return err
		}
		name := br.Info.BridgeID
		targets = append(targets, newTarget(name, conf))
	default:
		return errors.New("invalid target type")
	}

	for _, target := range targets {
		var wd string
		wd, err = os.Getwd()
		if err != nil {
			panic(err)
		}
		wd = filepath.Join(wd, "dump", args[0])
		parentFolder := ""
		if target.ParentFolder != "" {
			targetDir := filepath.Join(wd, target.ParentFolder)
			if err = os.MkdirAll(targetDir, 0o755); err != nil { // #nosec
				return err
			}
			log.Info().Msgf("created folder: %s", targetDir)
			parentFolder = target.ParentFolder
		}
		var js []byte
		if js, err = json.Marshal(target.Object); err != nil {
			return err
		}
		fpath := filepath.Join(wd, target.Name+".json")
		if parentFolder != "" {
			fpath = filepath.Join(wd, parentFolder, target.Name+".json")
		}
		if err = os.WriteFile(fpath, js, 0o666); err != nil {
			return err
		}
		// get current working directory

		log.Info().Msgf("dumped to: %s", fpath)
	}
	return nil

}

func cmdGetFullState(br *ziggy.Bridge, args []string) error {
	var err error
	var fullstate map[string]interface{}
	fullstate, err = br.GetFullState()
	if err != nil {
		return err
	}
	spew.Dump(fullstate)
	return nil
}

// cmdLoad imports a target JSON object and attempts to apply it to an existing object
func cmdLoad(br *ziggy.Bridge, args []string) error {
	var js []byte
	var err error
	switch len(args) {
	case 0, 1:
		return errors.New("not enough arguments")
	case 2:
		js, err = os.ReadFile(args[1])
	case 3:
		js, err = os.ReadFile(args[2])
	}
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("not enough arguments")
	}

	var target interface{}
	switch args[0] {
	case "light", "l":
		target, err = br.FindLight(args[1])
		if err != nil {
			return err
		}
		var l *huego.Light
		if err := json.Unmarshal(js, &l); err != nil {
			return err
		}
		if resp, err := br.UpdateLight(target.(*huego.Light).ID, *l); err != nil {
			return err
		} else {
			log.Info().Msgf("%v", resp)
		}
	case "group", "g":
		target, err = br.FindGroup(args[1])
		if err != nil {
			return err
		}
		var g *huego.Group
		if err := json.Unmarshal(js, &g); err != nil {
			return err
		}
		if resp, err := br.UpdateGroup(target.(*ziggy.HueGroup).ID, *g); err != nil {
			return err
		} else {
			log.Info().Msgf("%v", resp)
		}
	case "config", "conf", "cfg":
		var conf *huego.Config
		if err = json.Unmarshal(js, &conf); err != nil {
			return err
		}
		var resp *huego.Response
		if resp, err = br.UpdateConfig(conf); err != nil {
			return err
		}
		log.Info().Msgf("%v", resp)
	case "schedule":
		var sched *huego.Schedule
		if err = json.Unmarshal(js, &sched); err != nil {
			return err
		}
		var resp *huego.Response
		if resp, err = br.CreateSchedule(sched); err != nil {
			return err
		}
		log.Info().Msgf("%v", resp.Success)
	case "rule":
		var rule *huego.Rule
		if err = json.Unmarshal(js, &rule); err != nil {
			return err
		}
		var resp *huego.Response
		if resp, err = br.CreateRule(rule); err != nil {
			return err
		}
		log.Info().Msgf("%v", resp.Success)
	case "sensor":
		var sensor *huego.Sensor
		if err = json.Unmarshal(js, &sensor); err != nil {
			return err
		}
		var resp *huego.Response
		if resp, err = br.CreateSensor(sensor); err != nil {
			return err
		}
		log.Info().Msgf("%v", resp.Success)
	case "bridge":
		return errors.New("not implemented")
	default:
		return errors.New("invalid target type")
	}
	return nil
}

func cmdAdopt(br *ziggy.Bridge, args []string) error {
	defer ziggy.NeedsUpdate()
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
		log.Trace().Caller().Msgf("%v", spew.Sprint(l))
	}
	return nil
}

func cmdReboot(br *ziggy.Bridge, args []string) error {
	defer ziggy.NeedsUpdate()
	resp, err := br.UpdateConfig(&huego.Config{Reboot: true})
	if err != nil {
		return err
	}
	log.Info().Msgf("%v", resp)
	return nil
}
