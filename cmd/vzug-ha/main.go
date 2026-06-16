package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/tbuserdev/vzug-api/internal/config"
	"github.com/tbuserdev/vzug-api/internal/mqttbridge"
	"github.com/tbuserdev/vzug-api/internal/scheduler"
	"github.com/tbuserdev/vzug-api/internal/server"
	"github.com/tbuserdev/vzug-api/internal/state"
	"github.com/tbuserdev/vzug-api/internal/vzug"
)

type deviceClient interface {
	SetDisplayClock(ctx context.Context, visible bool) error
}

type app struct {
	cfg    config.Config
	device deviceClient
	store  *state.Store
	mqtt   *mqttbridge.Bridge
	logger *slog.Logger
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	device, err := vzug.New(vzug.Options{
		BaseURL:          cfg.BaseURL,
		AllowInsecureTLS: cfg.AllowInsecureTLS,
		Timeout:          cfg.HTTPTimeout,
		Retries:          cfg.RetryCount,
		RetryDelay:       cfg.RetryDelay,
	})
	if err != nil {
		logger.Error("failed to create V-ZUG client", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application := &app{
		cfg:    cfg,
		device: device,
		store:  state.New(),
		logger: logger,
	}

	if cfg.MQTTEnabled() {
		application.mqtt = mqttbridge.New(cfg.MQTT, application.store, application.SetDisplayClock, logger)
		if err := application.mqtt.Start(ctx); err != nil {
			logger.Error("failed to start MQTT bridge", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Warn("MQTT_BROKER is not set; Home Assistant discovery is disabled")
	}

	if _, err := scheduler.Start(ctx, scheduler.Config{
		Timezone:     cfg.Timezone,
		ShowSchedule: cfg.ShowSchedule,
		HideSchedule: cfg.HideSchedule,
	}, application.SetDisplayClock, logger); err != nil {
		logger.Error("failed to start scheduler", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           server.New(application),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("vzug-ha started", "port", cfg.Port, "base_url", cfg.BaseURL, "mqtt_enabled", cfg.MQTTEnabled())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}
	logger.Info("vzug-ha stopped")
}

func (a *app) SetDisplayClock(ctx context.Context, visible bool, action string) error {
	if err := a.device.SetDisplayClock(ctx, visible); err != nil {
		a.store.SetError(action, err)
		a.publishState()
		return err
	}
	a.store.SetVisible(visible, action)
	a.publishState()
	a.logger.Info("display clock state changed", "visible", visible, "action", action)
	return nil
}

func (a *app) Snapshot() state.Snapshot {
	return a.store.Snapshot()
}

func (a *app) CronDescription() string {
	return fmt.Sprintf(
		"Scheduled jobs (%s):\n- Show clock: %s\n- Hide clock: %s\n",
		a.cfg.Timezone,
		a.cfg.ShowSchedule,
		a.cfg.HideSchedule,
	)
}

func (a *app) publishState() {
	if a.mqtt == nil {
		return
	}
	if err := a.mqtt.PublishState(); err != nil {
		a.logger.Error("failed to publish MQTT state", "error", err)
	}
}
