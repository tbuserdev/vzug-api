package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

type Handler func(ctx context.Context, visible bool, action string) error

type Config struct {
	Timezone     string
	ShowSchedule string
	HideSchedule string
}

func Start(ctx context.Context, cfg Config, handler Handler, logger *slog.Logger) (*cron.Cron, error) {
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, err
	}
	jobs := cron.New(cron.WithLocation(location))
	if _, err := jobs.AddFunc(cfg.ShowSchedule, func() {
		run(ctx, handler, true, "schedule_show", logger)
	}); err != nil {
		return nil, err
	}
	if _, err := jobs.AddFunc(cfg.HideSchedule, func() {
		run(ctx, handler, false, "schedule_hide", logger)
	}); err != nil {
		return nil, err
	}
	jobs.Start()

	go func() {
		<-ctx.Done()
		stopCtx := jobs.Stop()
		<-stopCtx.Done()
	}()
	return jobs, nil
}

func run(parent context.Context, handler Handler, visible bool, action string, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err := handler(ctx, visible, action); err != nil {
		logger.Error("scheduled command failed", "action", action, "error", err)
	}
}
