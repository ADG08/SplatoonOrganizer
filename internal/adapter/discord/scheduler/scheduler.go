package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/bwmarrin/discordgo"
)

// Scheduler publishes the weekly availability messages.
type Scheduler struct {
	availSvc *appavailability.Service
	teams    *appteam.Service
	session  *discordgo.Session
}

// NewScheduler creates a scheduler.
func NewScheduler(
	availSvc *appavailability.Service,
	teams *appteam.Service,
	session *discordgo.Session,
) *Scheduler {
	return &Scheduler{
		availSvc: availSvc,
		teams:    teams,
		session:  session,
	}
}

// publishConcurrency bounds the teams published in parallel: each one renders its
// message (possibly an image) and calls Discord.
const publishConcurrency = 4

// checkEvery is how often the teams are checked, on the quarter of an hour: each team
// publishes on the hour of its own timezone, and some timezones are a half or a quarter
// of an hour off UTC (India, Nepal).
const checkEvery = 15 * time.Minute

// Start checks right away which teams reached their publication time (each picks its own
// in /help), to catch up on the publications missed while the bot was down, then every
// quarter of an hour until ctx is done. With force, that first check publishes for every
// team instead (for tests). Checks never overlap.
func (s *Scheduler) Start(ctx context.Context, force bool) {
	go func() {
		s.run(ctx, force)
		for {
			// Truncate rounds absolute time: the next quarter of an hour of every timezone.
			next := time.Now().Truncate(checkEvery).Add(checkEvery)
			select {
			case <-ctx.Done():
				slog.Debug("stopping availability scheduler")
				return
			case <-time.After(time.Until(next)):
				s.run(ctx, false)
			}
		}
	}()
}

// run publishes the weekly message of the active teams that configured a channel and
// are due (all of them with force). A team whose channel is out of reach is paused.
func (s *Scheduler) run(ctx context.Context, force bool) {
	teams, err := s.teams.ListWithChannel(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "weekly scheduler: list teams", "err", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, publishConcurrency)
	for _, t := range teams {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(ctx, weekly.Timeout)
			defer cancel()
			var err error
			if force {
				err = weekly.Publish(ctx, s.session, s.availSvc, t)
			} else {
				_, err = weekly.PublishDue(ctx, s.session, s.availSvc, t)
			}
			switch {
			case weekly.LostAccess(err):
				// Paused until the bot is back in the guild or /help sets a channel again.
				slog.WarnContext(ctx, "lost access to the weekly channel, pausing the team", "guild", t.GuildID, "team", t.Name, "err", err)
				if err := s.teams.SetActive(ctx, t.ID, false); err != nil {
					slog.ErrorContext(ctx, "pause team", "guild", t.GuildID, "team", t.Name, "err", err)
				}
			case err != nil:
				slog.ErrorContext(ctx, "publish weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
			}
		})
	}
	wg.Wait()
}
