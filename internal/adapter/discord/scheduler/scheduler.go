package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
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

// checkEvery is how often the guilds are checked, on the quarter of an hour: each guild
// publishes on the hour of its own timezone, and some timezones are a half or a quarter
// of an hour off UTC (India, Nepal).
const checkEvery = 15 * time.Minute

// Start checks right away which guilds reached their publication time (each picks its own
// in /help), to catch up on the publications missed while the bot was down, then every
// quarter of an hour until ctx is done. With force, that first check publishes in every
// guild instead (for tests). Checks never overlap.
func (s *Scheduler) Start(ctx context.Context, force bool) {
	go func() {
		s.run(ctx, force)
		for {
			// Truncate rounds absolute time: the next quarter of an hour of every timezone.
			next := time.Now().Truncate(checkEvery).Add(checkEvery)
			select {
			case <-ctx.Done():
				slog.Info("stopping availability scheduler")
				return
			case <-time.After(time.Until(next)):
				s.run(ctx, false)
			}
		}
	}()
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
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(ctx, weekly.Timeout)
			defer cancel()
			var err error
			if force {
				err = weekly.Publish(ctx, s.session, s.availSvc, cfg)
			} else {
				_, err = weekly.PublishDue(ctx, s.session, s.availSvc, cfg)
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
		})
	}
	wg.Wait()
}
