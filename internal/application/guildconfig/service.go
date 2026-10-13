package guildconfig

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
)

// Service implements guild configuration use cases.
type Service struct {
	repo Repository
}

// NewService creates a new guild config service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the guild configuration, with defaults when the guild has none.
func (s *Service) Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error) {
	return s.repo.Get(ctx, guildID)
}

// ListWithChannel returns the guilds that configured a channel for the weekly message.
func (s *Service) ListWithChannel(ctx context.Context) ([]*guildconfig.GuildConfig, error) {
	return s.repo.ListWithChannel(ctx)
}

// SetChannel updates the message channel for a guild.
func (s *Service) SetChannel(ctx context.Context, guildID, channelID string) error {
	return s.repo.SetChannel(ctx, guildID, channelID)
}

// SetRole updates the role to ping for a guild.
func (s *Service) SetRole(ctx context.Context, guildID, roleID string) error {
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

// SetSlotSettings updates how days are split into slots and how the weekly message is shown.
func (s *Service) SetSlotSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display) error {
	return s.repo.SetSlotSettings(ctx, guildID, layout, display)
}
