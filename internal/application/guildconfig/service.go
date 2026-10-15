package guildconfig

import (
	"context"
	"sync"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Service implements guild configuration use cases.
type Service struct {
	repo Repository

	// cache holds the configs already read: they are needed on nearly every interaction
	// and only change through this service (single bot process).
	cache sync.Map // guildID -> guildconfig.GuildConfig
	// writes serializes the changes, each one caching the config it leaves in the database.
	writes sync.Mutex
}

// NewService creates a new guild config service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the guild configuration, with defaults when the guild has none.
// The returned config is a copy the caller may modify.
func (s *Service) Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error) {
	if cfg, ok := s.cache.Load(guildID); ok {
		c := cfg.(guildconfig.GuildConfig)
		return &c, nil
	}
	cfg, err := s.repo.Get(ctx, guildID)
	if err != nil {
		return nil, err
	}
	// A change saved while this one was read cached the newer config: keep that one.
	cached, _ := s.cache.LoadOrStore(guildID, *cfg)
	c := cached.(guildconfig.GuildConfig)
	return &c, nil
}

// update saves a change, then caches the config as it now is in the database. Caching it
// (rather than forgetting it) keeps a read that started before the change from caching
// the old config.
func (s *Service) update(ctx context.Context, guildID string, write func() error) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := write(); err != nil {
		s.cache.Delete(guildID)
		return err
	}
	cfg, err := s.repo.Get(ctx, guildID)
	if err != nil {
		s.cache.Delete(guildID)
		return err
	}
	s.cache.Store(guildID, *cfg)
	return nil
}

// ListWithChannel returns the active guilds that configured a channel for the weekly message.
func (s *Service) ListWithChannel(ctx context.Context) ([]*guildconfig.GuildConfig, error) {
	return s.repo.ListWithChannel(ctx)
}

// SetChannel updates the message channel for a guild, and makes it active again.
func (s *Service) SetChannel(ctx context.Context, guildID, channelID string) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetChannel(ctx, guildID, channelID) })
}

// SetTimezone updates the timezone of the guild's hours.
func (s *Service) SetTimezone(ctx context.Context, guildID string, loc *time.Location) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetTimezone(ctx, guildID, loc) })
}

// SetActive turns the publication of the weekly message on or off for a guild, keeping
// its configuration.
func (s *Service) SetActive(ctx context.Context, guildID string, active bool) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetActive(ctx, guildID, active) })
}

// SetRole updates the role to ping for a guild.
func (s *Service) SetRole(ctx context.Context, guildID, roleID string) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetRole(ctx, guildID, roleID) })
}

// SetMessageSettings updates how days are split into slots, how the weekly message is
// shown and the language of the bot in the guild.
func (s *Service) SetMessageSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display, lang i18n.Lang) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetMessageSettings(ctx, guildID, layout, display, lang) })
}

// SetSchedule updates when the guild's weekly message is published.
func (s *Service) SetSchedule(ctx context.Context, guildID string, schedule availability.Schedule) error {
	return s.update(ctx, guildID, func() error { return s.repo.SetSchedule(ctx, guildID, schedule) })
}
