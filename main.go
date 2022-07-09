package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/amimof/huego"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/common"
	"git.tcp.direct/kayos/ziggs/config"
	"git.tcp.direct/kayos/ziggs/interactive"
	"git.tcp.direct/kayos/ziggs/ziggy"
)

var (
	log *zerolog.Logger
)

func init() {
	config.Init()
	log = config.StartLogger()
	log.Info().Msg("Logger started")
	if len(os.Args) < 1 {
		return
	}
}

func TurnAll(Known []*ziggy.Bridge, mode ziggy.ToggleMode) {
	for _, bridge := range Known {
		for _, l := range bridge.HueLights {
			go func(l *ziggy.HueLight) {
				l.Log().Debug().
					Str("caller", bridge.Host).
					Str("type", l.ProductName).
					Bool("on", l.IsOn()).Msg(l.ModelID)
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
				ziggy.Assert(ctx, l, mode)
				defer cancel()
			}(l)
		}
	}
}

func FindLights(ctx context.Context, c *ziggy.Bridge) error {
	log.Trace().Msg("looking for lights...")
	resp, err := c.FindLights()
	if err != nil {
		c.Log().Fatal().Err(err).Msg("FUBAR")
	}
	for str, inter := range resp.Success {
		c.Log().Trace().Interface(str, inter).Msg(" ")
	}
	var count = 0
	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			ls, err := c.GetLights()
			if err != nil {
				c.Log().Warn().Err(err).Msg(" ")
			}
			if len(ls) > count {
				count = len(ls)
				return nil
			}
		}
	}
}

func getNewSensors(known *ziggy.Bridge) {
	go known.FindSensors()
	Sensors, err := known.GetNewSensors()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get sensors")
	}
	if Sensors == nil {
		log.Fatal().Caller(1).Msg("nil")
	}
	for len(Sensors.Sensors) < 1 {
		Sensors, err = known.GetNewSensors()
		if err != nil {
			log.Error().Err(err).Msg("")
		}
		time.Sleep(2 * time.Second)
	}
	go log.Debug().Interface("sensors", Sensors).Msg("")
	selSensor(Sensors.Sensors)
}

func selSensor(Sensors []*huego.Sensor) huego.Sensor {
	p := promptui.Select{
		Label:        "Sensors",
		Items:        Sensors,
		CursorPos:    0,
		HideHelp:     false,
		HideSelected: false,
		Pointer:      common.ZiggsPointer,
	}
	i, s, e := p.Run()
	if e != nil {
		log.Error().Err(e).Msg("")
	}
	fmt.Printf("\nselected [%d] %s\n", i, s)
	return *Sensors[i]
}

func main() {
	var Known []*ziggy.Bridge
	var err error
	Known, err = ziggy.Setup()

	if err != nil {
		log.Fatal().Err(err).Msg("failed to get bridges")
	}

	for _, arg := range os.Args {
		switch arg {
		case "discover":

		case "on":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, ziggy.ToggleOn)
		case "off":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, ziggy.ToggleOff)
		case "rainbow":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, ziggy.ToggleRainbow)
		case "scan":
			log.Debug().Msg("executing " + arg)
			if len(os.Args) < 2 {
				for _, k := range Known {
					ctx := context.TODO()
					FindLights(ctx, k)
				}
			}
		case "shell":
			interactive.StartCLI()
		case "newsensor":
			getNewSensors(Known[0])
		case "sensors":
			sens, err := Known[0].GetSensors()
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			var sensptr []*huego.Sensor
			for _, s := range sens {
				sensptr = append(sensptr, &s)
			}
			selSensor(sensptr)
		}
	}

	done := make(chan struct{}, 1)
	<-done
}
