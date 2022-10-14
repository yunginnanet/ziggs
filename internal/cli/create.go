package cli

import (
	"errors"
	"strconv"

	"github.com/amimof/huego"

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
		for _, arg := range args {
			switch arg {
			case "group", name:
				continue
			case "-entertainment":
				groupType = "Entertainment"
				class = "Other"
				continue
			}
			_, err := strconv.Atoi(arg)
			if err != nil {
				return err
			}
			ids = append(ids, arg)
		}
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
