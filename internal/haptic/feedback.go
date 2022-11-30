package haptic

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
)

type EventClient struct {
	h             *http.Client
	subscriptions map[string]chan string
}

func NewEventClient() *EventClient {
	return &EventClient{
		h: &http.Client{
			Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			Timeout:   0,
		},
		subscriptions: make(map[string]chan string),
	}
}

func (c *EventClient) Subscribe(event string, ch chan string) {
	c.subscriptions[event] = ch
}

func (c *EventClient) Start(hueHost, hueKey string) error {
	req, err := http.NewRequest("GET", "https://"+hueHost+"/eventstream/clip/v2", nil)
	if err != nil {
		return err
	}
	req.Header.Add("hue-application-key", hueKey)
	req.Header.Add("Accept", "text/event-stream")
	resp, err := c.h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	xerox := bufio.NewScanner(resp.Body)
	for xerox.Scan() {
		if xerox.Err() != nil {
			return xerox.Err()
		}
		if ch, ok := c.subscriptions["*"]; ok {
			ch <- xerox.Text()
		}
		for term, ch := range c.subscriptions {
			if strings.Contains(xerox.Text(), term) {
				ch <- xerox.Text()
			}
		}
	}
	return io.EOF
}

func ListenToEvents(ctx context.Context, events chan string, hueHost, hueKey string) error {
	if hueHost == "" {
		return fmt.Errorf("hueHost is empty")
	}
	if hueKey == "" {
		return fmt.Errorf("hueKey is empty")
	}
	c := NewEventClient()
	ch := make(chan string, 5)
	c.Subscribe("*", ch)
	var errCh chan error
	go func() {
		errCh <- c.Start(hueHost, hueKey)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case event := <-ch:
			events <- event
		}
	}
}
