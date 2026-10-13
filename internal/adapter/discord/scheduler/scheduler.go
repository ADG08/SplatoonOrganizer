package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Scheduler runs the weekly availability posting job.
type Scheduler struct {
	availSvc    *appavailability.Service
	guildConfig *appguildconfig.Service
	session     *discordgo.Session
	guildID     string
	channelID   string // default from env
}

// NewScheduler creates a scheduler. channelID is the default channel (e.g. from env).
func NewScheduler(
	availSvc *appavailability.Service,
	guildConfig *appguildconfig.Service,
	session *discordgo.Session,
	guildID, channelID string,
) *Scheduler {
	return &Scheduler{
		availSvc:    availSvc,
		guildConfig: guildConfig,
		session:     session,
		guildID:     guildID,
		channelID:   channelID,
	}
}

// DefaultCronSchedule is the production schedule: every Monday at 10:00.
const DefaultCronSchedule = "0 10 * * MON"

// ErrChannelNotConfigured is returned when no channel is set (neither env nor /set-message-channel).
var ErrChannelNotConfigured = errors.New("channel not configured")

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

func (s *Scheduler) runWeeklyJob(ctx context.Context) error {
	channelID := s.channelID
	cfg := &guildconfig.GuildConfig{}

	if s.guildConfig != nil && s.guildID != "" {
		got, err := s.guildConfig.Get(ctx, s.guildID)
		if err != nil {
			log.Printf("get guild config: %v", err)
		} else if got != nil {
			cfg = got
			if cfg.ChannelID != "" {
				channelID = cfg.ChannelID
			}
		}
	}

	if channelID == "" {
		return ErrChannelNotConfigured
	}

	return handlers.PublishWeeklyMessage(ctx, s.session, s.availSvc, cfg, channelID)
}
