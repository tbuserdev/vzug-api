package vzug

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetDisplayClockCallsVZUGEndpoint(t *testing.T) {
	var gotCommand, gotValue string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCommand = r.URL.Query().Get("command")
		gotValue = r.URL.Query().Get("value")
		if r.URL.Path != "/hh" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.SetDisplayClock(context.Background(), true); err != nil {
		t.Fatalf("SetDisplayClock() error = %v", err)
	}
	if gotCommand != "setDisplayXclock" || gotValue != "true" {
		t.Fatalf("query command=%q value=%q", gotCommand, gotValue)
	}
}

func TestSetDisplayClockRetriesFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{
		BaseURL:    server.URL,
		Timeout:    time.Second,
		Retries:    1,
		RetryDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.SetDisplayClock(context.Background(), false); err != nil {
		t.Fatalf("SetDisplayClock() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}
