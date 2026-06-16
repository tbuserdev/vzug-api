package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/tbuserdev/vzug-api/internal/state"
)

type Controller interface {
	SetDisplayClock(ctx context.Context, visible bool, action string) error
	Snapshot() state.Snapshot
	CronDescription() string
}

func New(controller Controller) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, controller.Snapshot())
	})
	mux.HandleFunc("GET /show", func(w http.ResponseWriter, r *http.Request) {
		setVisible(w, r, controller, true, "http_show")
	})
	mux.HandleFunc("GET /hide", func(w http.ResponseWriter, r *http.Request) {
		setVisible(w, r, controller, false, "http_hide")
	})
	mux.HandleFunc("GET /toggle", func(w http.ResponseWriter, r *http.Request) {
		value, err := strconv.ParseBool(r.URL.Query().Get("value"))
		if err != nil {
			http.Error(w, "Missing or invalid 'value' parameter. Use true or false.", http.StatusBadRequest)
			return
		}
		setVisible(w, r, controller, value, "http_toggle")
	})
	mux.HandleFunc("GET /cron", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(controller.CronDescription()))
	})
	return mux
}

func setVisible(w http.ResponseWriter, r *http.Request, controller Controller, visible bool, action string) {
	if err := controller.SetDisplayClock(r.Context(), visible, action); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"visible": visible,
		"state":   state.Payload(visible),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		_, _ = fmt.Fprintf(w, `{"error":%q}`, err.Error())
	}
}
