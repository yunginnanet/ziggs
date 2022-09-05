package system

import (
	"context"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/muesli/termenv"
)

func TestCPULoadGradient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	grad, err := CPULoadGradient(ctx, "deepskyblue", "deeppink")
	if err != nil {
		t.Fatal(err)
	}
	p := termenv.ColorProfile()
	spew.Dump(p)
	s := termenv.String("yeet")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			col := <-grad
			fmt.Println(col.Hex() + ": " + s.Foreground(p.Color(col.Hex())).String())
		}
	}
}
