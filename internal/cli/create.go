package cli

import (
	"errors"
	"strconv"
	"strings"

	"github.com/yunginnanet/huego"

	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func cmdCreate(br *ziggy.Bridge, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	switch args[0] {
	case "group":
		var (
			name      = args[1]
			ids       []string
			groupType = "LightGroup"
			class     = ""
		)
		log.Debug().Msgf("creating group: %s", name)

		for i, arg := range args {
			switch arg {
			case "group", name:
				continue
			case "-entertainment":
				groupType = "Entertainment"
				class = "Other"
				log.Debug().Msgf("group type: %s", groupType)
				log.Debug().Msgf("group class: %s", class)
				continue
			}
			var seenMap = make(map[string]bool)
			if strings.Contains(arg, ",") {
				log.Debug().Msgf("found comma in arg %d, splitting argument by commas and remaking arg list", i)
				var newIDs []string
				newIDs = append(newIDs, strings.Split(arg, ",")...)
				log.Debug().Msgf("new args: %v", newIDs)
				for _, newArg := range newIDs {
					if _, err := strconv.Atoi(newArg); err != nil {
						return err
					}
					if seenMap[newArg] {
						continue
					}
					seenMap[newArg] = true
					ids = append(ids, newArg)
				}
				continue
			}

			if seenMap[arg] {
				continue
			}
			seenMap[arg] = true
			_, err := strconv.Atoi(arg)
			if err != nil {
				return err
			}
			log.Trace().Caller().Msgf("found light id: %s", arg)
			ids = append(ids, arg)
		}
		log.Debug().Msgf("light ids: %+s", ids)
		resp, err := br.CreateGroup(huego.Group{Name: name, Lights: ids, Type: groupType, Class: class})
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
