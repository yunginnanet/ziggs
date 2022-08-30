package system

import (
	"context"
	"time"

	"github.com/dhamith93/systats"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mazznoer/colorgrad"
)

func cpuLoad(ctx context.Context) (chan int, error) {
	syStats := systats.New()
	loadChan := make(chan int, 10)
	go func() {
		for {
			time.Sleep(250 * time.Millisecond)
			cpu, err := syStats.GetCPU()
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				loadChan <- cpu.LoadAvg
			}
		}
	}()
	return loadChan, nil
}

func CPULoadGradient(ctx context.Context, colors ...string) (chan colorful.Color, error) {
	grad, err := colorgrad.NewGradient().
		HtmlColors(colors...).
		Domain(0, 100).
		Build()
	if err != nil {
		return nil, err
	}
	load, err := cpuLoad(ctx)
	if err != nil {
		return nil, err
	}
	gradChan := make(chan colorful.Color, 10)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				gradChan <- grad.At(float64(<-load))
			}
		}
	}()
	return gradChan, nil
}
