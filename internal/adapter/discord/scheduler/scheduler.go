package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Scheduler publishes the weekly availability messages.
type Scheduler struct {
	availSvc    *appavailability.Service
	guildConfig *appguildconfig.Service
	session     *discordgo.Session
}

// NewScheduler creates a scheduler.
func NewScheduler(
	availSvc *appavailability.Service,
	guildConfig *appguildconfig.Service,
	session *discordgo.Session,
) *Scheduler {
	return &Scheduler{
		availSvc:    availSvc,
		guildConfig: guildConfig,
		session:     session,
	}
}

// publishConcurrency bounds the guilds published in parallel: each one renders its
// message (possibly an image) and calls Discord.
const publishConcurrency = 4

// checkSchedule runs every quarter of an hour: each guild publishes on the hour of its own
// timezone, and some timezones are a half or a quarter of an hour off UTC (India, Nepal).
const checkSchedule = "*/15 * * * *"

// Start checks every quarter of an hour which guilds reached their publication time (each
// guild picks its own in /help). It checks once right away too, to catch up on the
// publications missed while the bot was down; with force, that first run publishes in
// every guild instead (for tests).
func (s *Scheduler) Start(ctx context.Context, force bool) error {
	c := cron.New(cron.WithLocation(time.UTC))
	if _, err := c.AddFunc(checkSchedule, func() { s.run(ctx, false) }); err != nil {
		return fmt.Errorf("adding cron job: %w", err)
	}
	c.Start()
	go s.run(ctx, force)

	go func() {
		<-ctx.Done()
		slog.Info("stopping availability scheduler")
		c.Stop()
	}()

	return nil
}

// run publishes the weekly message of the active guilds that configured a channel and
// are due (all of them with force). A guild whose channel is out of reach is paused.
func (s *Scheduler) run(ctx context.Context, force bool) {
	cfgs, err := s.guildConfig.ListWithChannel(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "weekly scheduler: list guild configs", "err", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, publishConcurrency)
	for _, cfg := range cfgs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			ctx, cancel := context.WithTimeout(ctx, weekly.Timeout)
			defer cancel()
			due, err := weekly.Due(ctx, s.availSvc, cfg)
			if err == nil && (due || force) {
				err = weekly.Publish(ctx, s.session, s.availSvc, cfg)
			}
			switch {
			case weekly.LostAccess(err):
				// Paused until the bot is back in the guild or /help sets a channel again.
				slog.WarnContext(ctx, "lost access to the weekly channel, pausing the guild", "guild", cfg.GuildID, "err", err)
				if err := s.guildConfig.SetActive(ctx, cfg.GuildID, false); err != nil {
					slog.ErrorContext(ctx, "pause guild", "guild", cfg.GuildID, "err", err)
				}
			case err != nil:
				slog.ErrorContext(ctx, "publish weekly message", "guild", cfg.GuildID, "err", err)
			}
		}()
	}
	wg.Wait()
}
