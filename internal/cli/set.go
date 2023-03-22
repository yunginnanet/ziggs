package cli

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/amimof/huego"

	"git.tcp.direct/kayos/ziggs/internal/common"
	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

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
	Scene(string) error
	Effect(string) error
}

var ErrNotEnoughArguments = errors.New("not enough arguments")

func cmdSet(bridge *ziggy.Bridge, args []string) error {
	if len(args) < 3 {
		return ErrNotEnoughArguments
	}

	type (
		action func() error
	)

	var (
		groupMap     map[string]*ziggy.HueGroup
		lightMap     map[string]*ziggy.HueLight
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
			groupMap = ziggy.GetGroupMap()
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
			log.Trace().Caller().Msgf("color, args: %v", args)
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
				return ErrNotEnoughArguments
			}
			argHead++
			newHue, numErr := strconv.Atoi(strings.TrimSpace(args[argHead]))
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
				return ErrNotEnoughArguments
			}
			argHead++
			newSat, numErr := strconv.Atoi(strings.TrimSpace(args[argHead]))
			if numErr != nil {
				return fmt.Errorf("given saturation is not a valid number: %v", numErr)
			}
			actions = append(actions, func() error {
				err := target.Sat(uint8(newSat))
				if err != nil {
					err = fmt.Errorf("failed to set saturation: %w", err)
				}
				return err
			})
		case "effect", "e":
			if len(args) == argHead-1 {
				return ErrNotEnoughArguments
			}
			argHead++
			newEffect := strings.TrimSpace(args[argHead])
			actions = append(actions, func() error {
				err := target.Effect(newEffect)
				if err != nil {
					err = fmt.Errorf("failed to set effect: %w", err)
				}
				return err
			})
		case "temperature", "temp":
			if len(args) == argHead-1 {
				return ErrNotEnoughArguments
			}
			argHead++
			newTemp, numErr := strconv.Atoi(strings.TrimSpace(args[argHead]))
			if numErr != nil || newTemp > 500 || newTemp < 153 {
				terr := fmt.Errorf("given temperature is not a valid number: %w", numErr)
				if numErr == nil {
					terr = fmt.Errorf("temperature must be greater than 153 and less than 500")
				}
				return terr
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
			go func() {
				if err := cpuInit(args[argHead], bridge, target); err != nil {
					log.Error().Err(err).Msg("cpu init failed")
				}
			}()
			log.Info().Msg("cpu load lighting started")
			return nil
		case "scene", "sc":
			if len(args) == argHead-1 {
				return ErrNotEnoughArguments
			}
			argHead++
			if argHead > len(args)-1 {
				return ErrNotEnoughArguments
			}
			targetScene := strings.TrimSpace(args[argHead])
			actions = append(actions, func() error {
				err := target.Scene(targetScene)
				if err != nil {
					targetScene = ziggy.GetSceneMap()[targetScene].ID
					err = target.Scene(targetScene)
					if err != nil {
						err = fmt.Errorf("failed to set scene: %w", err)
					}
				}
				return err
			})

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
	tg, tgok := target.(*ziggy.HueGroup)
	tl, tlok := target.(*ziggy.HueLight)
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
