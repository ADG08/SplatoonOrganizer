package guildconfig

import (
	"context"
	"sync"

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
	s.cache.Store(guildID, *cfg)
	return cfg, nil
}

// ListWithChannel returns the guilds that configured a channel for the weekly message.
func (s *Service) ListWithChannel(ctx context.Context) ([]*guildconfig.GuildConfig, error) {
	return s.repo.ListWithChannel(ctx)
}

// SetChannel updates the message channel for a guild.
func (s *Service) SetChannel(ctx context.Context, guildID, channelID string) error {
	defer s.cache.Delete(guildID)
	return s.repo.SetChannel(ctx, guildID, channelID)
}

// SetRole updates the role to ping for a guild.
func (s *Service) SetRole(ctx context.Context, guildID, roleID string) error {
	defer s.cache.Delete(guildID)
	return s.repo.SetRole(ctx, guildID, roleID)
}

// SlotLayout returns how days are split into slots for a guild (half-day by default).
func (s *Service) SlotLayout(ctx context.Context, guildID string) (availability.Layout, error) {
	cfg, err := s.Get(ctx, guildID)
	if err != nil {
		return availability.Layout{}, err
	}
	return cfg.SlotLayout, nil
}

// SetMessageSettings updates how days are split into slots, how the weekly message is
// shown and the language of the bot in the guild.
func (s *Service) SetMessageSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display, lang i18n.Lang) error {
	defer s.cache.Delete(guildID)
	return s.repo.SetMessageSettings(ctx, guildID, layout, display, lang)
}
