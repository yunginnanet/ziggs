package haptic

import (
	"context"
	"os"
	"testing"
)

func TestListenToEvents(t *testing.T) {
	if os.Getenv("ZIGGS_CI_KEY") == "" {
		t.Skip("skipping test requiring ZIGGS_CI_KEY environment variable.")
	}
	events := make(chan string, 5)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		errCh <- ListenToEvents(ctx, events, os.Getenv("ZIGGS_CI_HOST"), os.Getenv("ZIGGS_CI_KEY"))
	}()
	for {
		select {
		case err := <-errCh:
			cancel()
			t.Fatal(err)
		case <-ctx.Done():
			cancel()
			t.Log("context cancelled")
			return
		case in := <-events:
			t.Log(in)
		}
	}
}
