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
