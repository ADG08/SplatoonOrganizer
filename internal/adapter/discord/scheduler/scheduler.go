package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"

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

// Start checks every hour, on the hour, which guilds reached their publication time
// (each guild picks its own in /help). It checks once right away too, to catch up on the
// publications missed while the bot was down; with force, that first run publishes in
// every guild instead (for tests).
func (s *Scheduler) Start(ctx context.Context, force bool) error {
	c := cron.New(cron.WithLocation(s.availSvc.Location()))
	if _, err := c.AddFunc("@hourly", func() { s.run(ctx, false) }); err != nil {
		return fmt.Errorf("adding cron job: %w", err)
	}
	c.Start()
	go s.run(ctx, force)

	go func() {
		<-ctx.Done()
		log.Println("stopping availability scheduler...")
		c.Stop()
	}()

	return nil
}

// run publishes the weekly message of the guilds that configured a channel and are due
// (all of them with force).
func (s *Scheduler) run(ctx context.Context, force bool) {
	cfgs, err := s.guildConfig.ListWithChannel(ctx)
	if err != nil {
		log.Printf("weekly scheduler: list guild configs: %v", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, publishConcurrency)
	for _, cfg := range cfgs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			due, err := weekly.Due(ctx, s.availSvc, cfg)
			if err == nil && (due || force) {
				err = weekly.Publish(ctx, s.session, s.availSvc, cfg)
			}
			if err != nil {
				log.Printf("guild %s: publish weekly message: %v", cfg.GuildID, err)
			}
		}()
	}
	wg.Wait()
}
