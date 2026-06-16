package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tbuserdev/vzug-api/internal/state"
)

type fakeController struct {
	store *state.Store
	err   error
}

func (f fakeController) SetDisplayClock(ctx context.Context, visible bool, action string) error {
	if f.err != nil {
		return f.err
	}
	f.store.SetVisible(visible, action)
	return nil
}

func (f fakeController) Snapshot() state.Snapshot {
	return f.store.Snapshot()
}

func (f fakeController) CronDescription() string {
	return "cron"
}

func TestToggleEndpoint(t *testing.T) {
	controller := fakeController{store: state.New()}
	req := httptest.NewRequest(http.MethodGet, "/toggle?value=true", nil)
	res := httptest.NewRecorder()

	New(controller).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	if !controller.store.Snapshot().Visible {
		t.Fatal("visible state was not updated")
	}
}

func TestToggleEndpointRejectsInvalidValue(t *testing.T) {
	controller := fakeController{store: state.New()}
	req := httptest.NewRequest(http.MethodGet, "/toggle?value=wat", nil)
	res := httptest.NewRecorder()

	New(controller).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestCommandFailureReturnsBadGateway(t *testing.T) {
	controller := fakeController{store: state.New(), err: errors.New("device down")}
	req := httptest.NewRequest(http.MethodGet, "/show", nil)
	res := httptest.NewRecorder()

	New(controller).ServeHTTP(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", res.Code)
	}
}
