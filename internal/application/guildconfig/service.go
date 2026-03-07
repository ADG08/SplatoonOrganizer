package guildconfig

import (
	"context"

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

// Get returns the guild configuration. Returns empty config if not found.
func (s *Service) Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error) {
	cfg, err := s.repo.Get(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &guildconfig.GuildConfig{}, nil
	}
	return cfg, nil
}

// SetChannel updates the message channel for a guild.
func (s *Service) SetChannel(ctx context.Context, guildID, channelID string) error {
	return s.repo.SetChannel(ctx, guildID, channelID)
}

// SetRole updates the role to ping for a guild.
func (s *Service) SetRole(ctx context.Context, guildID, roleID string) error {
	return s.repo.SetRole(ctx, guildID, roleID)
}
