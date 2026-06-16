package mqttbridge

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestParseSwitchPayload(t *testing.T) {
	cases := map[string]bool{
		"ON":    true,
		"true":  true,
		"1":     true,
		"OFF":   false,
		"false": false,
		"0":     false,
	}
	for payload, want := range cases {
		got, err := parseSwitchPayload(payload)
		if err != nil {
			t.Fatalf("parseSwitchPayload(%q) error = %v", payload, err)
		}
		if got != want {
			t.Fatalf("parseSwitchPayload(%q) = %v, want %v", payload, got, want)
		}
	}
	if _, err := parseSwitchPayload("maybe"); err == nil {
		t.Fatal("parseSwitchPayload(maybe) expected error")
	}
}

func TestRunCommandsSerializesFIFO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var order []bool
	active := 0
	maxActive := 0
	done := make(chan struct{})

	bridge := &Bridge{
		ctx:      ctx,
		commands: make(chan command, 2),
		logger:   slog.Default(),
		handler: func(ctx context.Context, visible bool, action string) error {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()

			if visible {
				time.Sleep(20 * time.Millisecond)
			}

			mu.Lock()
			order = append(order, visible)
			active--
			if len(order) == 2 {
				close(done)
			}
			mu.Unlock()
			return nil
		},
	}

	go bridge.runCommands()
	bridge.commands <- command{visible: true}
	bridge.commands <- command{visible: false}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for commands")
	}

	mu.Lock()
	defer mu.Unlock()
	if maxActive != 1 {
		t.Fatalf("maxActive = %d, want 1", maxActive)
	}
	if len(order) != 2 || !order[0] || order[1] {
		t.Fatalf("order = %v, want [true false]", order)
	}
}
