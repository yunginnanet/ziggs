package system

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.tcp.direct/kayos/common/entropy"
	"github.com/davecgh/go-spew/spew"
	"github.com/muesli/termenv"
)

var grindSet = int64(0)
var grindOnce = &sync.Once{}

func grind(ctx context.Context) {
	if atomic.LoadInt64(&grindSet) > 50 {
		return
	}
	var cancel context.CancelFunc
	grindOnce.Do(
		func() {
			ctx, cancel = context.WithDeadline(ctx, time.Now().Add(10*time.Second))
		})
	atomic.AddInt64(&grindSet, 1)
	time.Sleep(time.Duration(5000-(entropy.RNG(int(atomic.LoadInt64(&grindSet))*7500))) * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			go grind(ctx)
			go io.Copy(io.Discard, strings.NewReader(entropy.RandStr(9999999999)))
		}
	}
}

var once = &sync.Once{}

func TestCPULoadGradient(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(15*time.Second))
	defer cancel()
	grad, err := CPULoadGradient(ctx, "deepskyblue", "deeppink")
	if err != nil {
		t.Fatal(err)
	}
	p := termenv.ColorProfile()
	spew.Dump(p)
	s := termenv.String("yeet")
	var count = 0
	for {
		select {
		case <-ctx.Done():
			t.Logf("done with gradient test")
			return
		case col := <-grad:
			fmt.Println(col.Hex() + ": " + s.Foreground(p.Color(col.Hex())).String())
			count++
		default:
			once.Do(func() {
				fmt.Println("generating CPU load")
				go grind(context.Background())
			})
		}
	}
}

func TestCoreLoadHue(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(20*time.Second))
	defer cancel()
	huint, err := CoreLoadHue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case <-ctx.Done():
			t.Logf("done with core hue test")
			return
		case hue := <-huint:
			fmt.Println(hue)
		default:
		}
	}
}
