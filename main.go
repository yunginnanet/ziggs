package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"

	"git.tcp.direct/kayos/ziggs/config"
	"git.tcp.direct/kayos/ziggs/interactive"
	"git.tcp.direct/kayos/ziggs/lights"
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

func TurnAll(Known []*lights.Bridge, mode lights.ToggleMode) {
	for _, bridge := range Known {
		for _, l := range bridge.HueLights {
			go func(l *lights.HueLight) {
				l.Log().Debug().
					Str("caller", bridge.Host).
					Str("type", l.ProductName).
					Bool("on", l.IsOn()).Msg(l.ModelID)
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
				lights.Assert(ctx, l, mode)
				defer cancel()
			}(l)
		}
	}
}

func FindLights(ctx context.Context, c *lights.Bridge) error {
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

func main() {
	var Known []*lights.Bridge
	var err error
	Known, err = lights.Setup()

	if err != nil {
		log.Fatal().Err(err).Msg("failed to get bridges")
	}

	for _, arg := range os.Args {
		switch arg {
		case "discover":

		case "on":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, lights.ToggleOn)
		case "off":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, lights.ToggleOff)
		case "rainbow":
			log.Debug().Msg("turning all " + arg)
			TurnAll(Known, lights.ToggleRainbow)
		case "scan":
			log.Debug().Msg("executing " + arg)
			if len(os.Args) < 2 {
				for _, k := range Known {
					ctx := context.TODO()
					FindLights(ctx, k)
				}
			}
		default:
			interactive.StartCLI()
		}
	}

	done := make(chan struct{}, 1)
	<-done
}
