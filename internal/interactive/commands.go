package interactive

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/amimof/huego"
	"github.com/davecgh/go-spew/spew"
	"github.com/lucasb-eyer/go-colorful"

	"git.tcp.direct/kayos/ziggs/internal/common"
	"git.tcp.direct/kayos/ziggs/internal/system"
	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

var (
	cpuOn      = false
	cpuCtx     context.Context
	cpuCancel  context.CancelFunc
	cpuLastCol string
	cpuLastHue       = make(map[int]uint16)
	brightness uint8 = 0
)

var bridgeCMD = map[string]reactor{
	"schedules": cmdSchedules,
	"senors":    cmdSensors,
	"lights":    cmdLights,
	"groups":    cmdGroups,
	"delete":    cmdDelete,
	"rename":    cmdRename,
	"create":    cmdCreate,
	"rules":     cmdRules,
	"scan":      cmdScan,
	"set":       cmdSet,
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
		if strings.ToLower(input) == "y" {
			return true
		}
		return false
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

func cmdCreate(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	switch args[0] {
	case "group":
		var name = args[1]
		var ids []string
		for _, arg := range args {
			if arg == "group" || arg == name {
				continue
			}
			_, err := strconv.Atoi(arg)
			if err != nil {
				return err
			}
			ids = append(ids, arg)
		}
		resp, err := br.CreateGroup(huego.Group{Name: name, Lights: ids})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "schedule":
		resp, err := br.CreateSchedule(&huego.Schedule{Name: args[1]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "rule":
		resp, err := br.CreateRule(&huego.Rule{Name: args[1]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	case "sensor":
		resp, err := br.CreateSensor(&huego.Sensor{Name: args[1]})
		if err != nil {
			return err
		}
		log.Info().Msgf("response: %v", resp)
	default:
		return errors.New("invalid target type")
	}
	return nil
}

type cmdTarget interface {
	On() error
	Off() error
	Bri(uint8) error
	Ct(uint16) error
	Hue(uint16) error
	Sat(uint8) error
	Col(color.Color) error
	SetState(huego.State) error
	Alert(string) error
}

func cmdSet(bridge *ziggy.Bridge, args []string) error {
	if len(args) < 3 {
		return errors.New("not enough arguments")
	}

	type (
		action func() error
	)

	var (
		groupMap     map[string]*huego.Group
		lightMap     map[string]*huego.Light
		actions      []action
		currentState *huego.State
		argHead      = -1
		target       cmdTarget
	)

	for range args {
		argHead++
		if len(args) <= argHead {
			break
		}
		log.Trace().Int("argHead", argHead).Msg(args[argHead])
		switch args[argHead] {
		case "group", "g":
			var err error
			groupMap, err = ziggy.GetGroupMap()
			if err != nil {
				return err
			}
			if len(args) <= argHead-1 {
				return errors.New("no group specified")
			}
			argHead++
			g, ok := groupMap[strings.TrimSpace(args[argHead])]
			if !ok {
				return fmt.Errorf("group %s not found (argHead: %d)", args[argHead], argHead)
			}
			log.Trace().Str("group", g.Name).Msgf("found group %s via args[%d]",
				args[argHead], argHead,
			)
			target = g
		case "light", "l":
			lightMap = ziggy.GetLightMap()
			if len(args) <= argHead-1 {
				return errors.New("no light specified")
			}
			argHead++
			l, ok := lightMap[strings.TrimSpace(args[argHead])]
			if !ok {
				return fmt.Errorf("light %s not found (argHead: %d)", args[argHead], argHead)
			}
			if extraDebug {
				log.Trace().Str("group", l.Name).Msgf("found light %s via args[%d]",
					args[argHead], argHead)
			}
			target = l
		case "on":
			actions = append(actions, target.On)
		case "off":
			actions = append(actions, target.Off)
		case "brightness--", "dim":
			actions = append(actions, func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				err := target.Bri(currentState.Bri - 5)
				if err != nil {
					err = fmt.Errorf("couldn't lower brightness: %w", err)
				}
				return err
			})
		case "brightness++", "brighten":
			actions = append(actions, func() error {
				if currentState == nil {
					return fmt.Errorf("no state found")
				}
				err := target.Bri(currentState.Bri + 5)
				if err != nil {
					err = fmt.Errorf("couldn't raise brightness: %w", err)
				}
				return err
			})
		case "brightness", "bri":
			if len(args) == argHead-1 {
				return errors.New("no brightness specified")
			}
			argHead++
			newBrightness, numErr := strconv.Atoi(args[argHead])
			if numErr != nil {
				return fmt.Errorf("given brightness is not a number: %w", numErr)
			}
			actions = append(actions, func() error {
				err := target.Bri(uint8(newBrightness))
				if err != nil {
					err = fmt.Errorf("failed to set brightness: %w", err)
				}
				return err
			})
		case "color":
			if len(args) == argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newcolor, err := common.ParseHexColorFast(args[argHead])
			if err != nil {
				return err
			}
			actions = append(actions, func() error {
				colErr := target.Col(newcolor)
				if colErr != nil {
					colErr = fmt.Errorf("failed to set color: %w", colErr)
				}
				return colErr
			})
		case "hue", "h":
			if len(args) == argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newHue, numErr := strconv.Atoi(args[argHead])
			if numErr != nil || newHue > 65535 || newHue < 0 {
				return fmt.Errorf("given hue is not a valid number: %w", numErr)
			}
			actions = append(actions, func() error {
				err := target.Hue(uint16(newHue))
				if err != nil {
					err = fmt.Errorf("failed to set hue: %w", err)
				}
				return err
			})
		case "saturation", "sat":
			if len(args) == argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newSat, numErr := strconv.Atoi(args[argHead])
			if numErr != nil || newSat > 255 || newSat < 0 {
				return fmt.Errorf("given saturation is not a valid number: %w", numErr)
			}
			actions = append(actions, func() error {
				err := target.Sat(uint8(newSat))
				if err != nil {
					err = fmt.Errorf("failed to set saturation: %w", err)
				}
				return err
			})
		case "temperature", "temp":
			if len(args) == argHead-1 {
				return errors.New("not enough arguments")
			}
			argHead++
			newTemp, numErr := strconv.Atoi(args[argHead])
			if numErr != nil || newTemp > 500 || newTemp < 153 {
				return fmt.Errorf("given temperature is not a valid number: %w", numErr)
			}
			actions = append(actions, func() error {
				err := target.Ct(uint16(newTemp))
				if err != nil {
					err = fmt.Errorf("failed to set temperature: %w", err)
				}
				return err
			})
		case "alert":
			actions = append(actions, func() error {
				alErr := target.Alert("select")
				if alErr != nil {
					alErr = fmt.Errorf("failed to turn on alert: %w", alErr)
				}
				return alErr
			})
		case "cpu", "cpu2":
			switch cpuOn {
			case false:
				var load chan colorful.Color
				var coreLoad chan uint16
				var err error
				cpuCtx, cpuCancel = context.WithCancel(context.Background())
				if args[argHead] == "cpu" {
					load, err = system.CPULoadGradient(cpuCtx,
						"cornflowerblue", "deepskyblue", "gold", "deeppink", "darkorange", "red")
					if err != nil {
						return err
					}
				} else {
					coreLoad, err = system.CoreLoadHue(cpuCtx)
					if err != nil {
						return err
					}
				}
				log.Info().Msg("turning CPU load lights on for ")
				go func(cpuTarget cmdTarget) {
					var head = 0
					cpuOn = true
					defer func() {
						cpuOn = false
					}()
					var lights []*huego.Light
					for _, l := range cpuTarget.(*huego.Group).Lights {
						lint, _ := strconv.Atoi(l)
						lptr, err := bridge.GetLight(lint)
						if err != nil {
							log.Error().Err(err).Msg("failed to get light")
							continue
						}
						lights = append(lights, lptr)
					}
					for {
						select {
						case <-cpuCtx.Done():
							cpuOn = false
							return
						case clr := <-load:
							time.Sleep(2 * time.Second)
							if clr.Hex() == cpuLastCol {
								continue
							}
							cpuLastCol = clr.Hex()
							log.Trace().Msgf("CPU load color: %v", clr.Hex())
							cHex, cErr := common.ParseHexColorFast(clr.Hex())
							if cErr != nil {
								log.Error().Err(cErr).Msg("failed to parse color")
								continue
							}
							if brightness != 0 {
								_ = cpuTarget.Bri(brightness)
							}
							colErr := cpuTarget.Col(cHex)
							if colErr != nil {
								log.Error().Err(colErr).Msg("failed to set color")
								time.Sleep(3 * time.Second)
								continue
							}
						case hue := <-coreLoad:
							if head > len(lights)-1 {
								head = 0
							}
							if hue == cpuLastHue[head] {
								continue
							}
							time.Sleep(1 * time.Second)
							cpuLastHue[head] = hue
							log.Trace().Msgf("CPU load hue: %v", hue)
							target := lights[head]
							hueErr := target.Hue(65535-hue)
							head++
							if hueErr != nil {
								log.Error().Err(hueErr).Msg("failed to set hue")
								time.Sleep(3 * time.Second)
								continue
							}
						}
					}
				}(target)
				return nil
			case true:
				log.Info().Msg("turning CPU load lights off")
				cpuCancel()
				cpuOn = false
				return nil
			}
		default:
			return fmt.Errorf("unknown argument: " + args[argHead])
		}
	}
	if actions == nil {
		return errors.New("no action specified")
	}
	if target == nil {
		return errors.New("no target specified")
	}
	tg, tgok := target.(*huego.Group)
	tl, tlok := target.(*huego.Light)
	switch {
	case tgok:
		currentState = tg.State
	case tlok:
		currentState = tl.State
	default:
		return errors.New("unknown target")
	}
	log.Trace().Msgf("current state: %v", currentState)
	for d, act := range actions {
		log.Trace().Msgf("running action %d", d)
		err := act()
		if err != nil {
			return err
		}
		switch {
		case tgok:
			currentState = tg.State
		case tlok:
			currentState = tl.State
		}
		log.Trace().Msgf("new state: %v", currentState)
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
