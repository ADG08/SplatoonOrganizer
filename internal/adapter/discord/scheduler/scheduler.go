package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Scheduler runs the weekly availability posting job.
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

// DefaultCronSchedule is the production schedule: every Monday at 10:00.
const DefaultCronSchedule = "0 10 * * MON"

// publishConcurrency bounds the guilds published in parallel: each one renders its
// message (possibly an image) and calls Discord.
const publishConcurrency = 4

// Start starts the cron scheduler.
func (s *Scheduler) Start(ctx context.Context, schedule string) error {
	if schedule == "" {
		schedule = DefaultCronSchedule
	}

	loc := s.availSvc.Location()
	c := cron.New(cron.WithLocation(loc))

	_, err := c.AddFunc(schedule, func() {
		if err := s.runWeeklyJob(ctx); err != nil {
			log.Printf("weekly scheduler job failed: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("adding cron job: %w", err)
	}

	c.Start()

	go func() {
		<-ctx.Done()
		log.Println("stopping availability scheduler...")
		c.Stop()
	}()

	return nil
}

// RunWeeklyNow runs the weekly job once (for testing or manual trigger).
func (s *Scheduler) RunWeeklyNow(ctx context.Context) error {
	return s.runWeeklyJob(ctx)
}

// runWeeklyJob posts the new weekly message in every guild that configured a channel.
func (s *Scheduler) runWeeklyJob(ctx context.Context) error {
	cfgs, err := s.guildConfig.ListWithChannel(ctx)
	if err != nil {
		return fmt.Errorf("list guild configs: %w", err)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, publishConcurrency)
	for _, cfg := range cfgs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			if err := handlers.PublishWeeklyMessage(ctx, s.session, s.availSvc, cfg); err != nil {
				log.Printf("guild %s: publish weekly message: %v", cfg.GuildID, err)
			}
		}()
	}
	wg.Wait()
	return nil
}
