package cli

import (
	"context"
	"strconv"
	"time"

	"github.com/yunginnanet/huego"
	"github.com/lucasb-eyer/go-colorful"

	"git.tcp.direct/kayos/ziggs/internal/common"
	"git.tcp.direct/kayos/ziggs/internal/system"
	"git.tcp.direct/kayos/ziggs/internal/ziggy"
)

func cpuInit(argVal string, bridge *ziggy.Bridge, cpuTarget cmdTarget) error {
	if cpuOn {
		log.Info().Msg("turning CPU load lights off")
		cpuCancel()
		cpuOn = false
		return nil
	}
	var load chan colorful.Color
	var coreLoad chan uint16
	var err error
	cpuCtx, cpuCancel = context.WithCancel(context.Background())
	if argVal == "cpu" {
		load, err = system.CPULoadGradient(cpuCtx,
			"cornflowerblue", "deepskyblue", "#FFD700", "deeppink", "darkorange", "red", "#FFFFFF")
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

	var head = 0
	cpuOn = true
	defer func() {
		cpuOn = false
	}()
	var lights []*huego.Light
	for _, l := range cpuTarget.(*ziggy.HueGroup).Lights {
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
			return nil
		case clr := <-load:
			time.Sleep(750 * time.Millisecond)
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
			time.Sleep(750 * time.Millisecond)
			cpuLastHue[head] = hue
			// log.Trace().Msgf("CPU load hue: %v", hue)
			target := lights[head]
			newh := 65000 - hue
			if newh < 1 {
				newh = 1
			}
			hueErr := target.Hue(newh)
			if hueErr != nil {
				log.Error().Err(hueErr).Msg("failed to set hue")
				time.Sleep(3 * time.Second)
				continue
			}
			head++
		}
	}
}
