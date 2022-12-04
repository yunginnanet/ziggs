package system

import (
	"context"
	"time"

	"github.com/dhamith93/systats"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mazznoer/colorgrad"
)

var syStats = systats.New()

func CPULoad(ctx context.Context) (chan int, error) {
	loadChan := make(chan int)
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
			case loadChan <- cpu.LoadAvg:
				//
			default:
				//
			}
		}
	}()
	return loadChan, nil
}

func CoreLoads(ctx context.Context) (perCoreLoad chan uint16, coreCount int, err error) {
	perCoreLoad = make(chan uint16)
	var c systats.CPU
	c, err = syStats.GetCPU()
	if err != nil {
		return
	}
	coreCount = len(c.CoreAvg)
	go func() {
		for {
			c, err = syStats.GetCPU()
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
				for _, core := range c.CoreAvg {
					if core == 0 {
						continue
					}
					if core > 100 {
						continue
					}
					perCoreLoad <- uint16(core)
					time.Sleep(250 * time.Millisecond)
				}
			}
		}
	}()
	return
}

func CPULoadGradient(ctx context.Context, colors ...string) (chan colorful.Color, error) {
	grad, err := colorgrad.NewGradient().
		HtmlColors(colors...).
		Domain(0, 100).
		Build()
	if err != nil {
		return nil, err
	}
	load, err := CPULoad(ctx)
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

func CoreLoadHue(ctx context.Context) (chan uint16, error) {
	cores, coreCount, err := CoreLoads(ctx)
	if err != nil {
		return nil, err
	}
	hueChan := make(chan uint16)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case core := <-cores:
				rd := uint16(float64(core) / float64(coreCount) * 360)
				hueChan <- rd * 650
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	return hueChan, nil
}
