package guild

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Service implements the use cases of a guild as a whole.
type Service struct {
	repo Repository
}

// NewService creates a new guild service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the guild, empty when nothing was saved for it.
func (s *Service) Get(ctx context.Context, guildID string) (*guild.Guild, error) {
	return s.repo.Get(ctx, guildID)
}

// SetLanguage saves the language picked for what isn't a team's.
func (s *Service) SetLanguage(ctx context.Context, guildID string, lang i18n.Lang) error {
	return s.repo.SetLanguage(ctx, guildID, lang)
}

// SetWelcome remembers the message posted when the bot joined the guild.
func (s *Service) SetWelcome(ctx context.Context, guildID string, welcome guild.Welcome) error {
	return s.repo.SetWelcome(ctx, guildID, welcome)
}

// ForgetWelcome forgets the guild's welcome message: the bot left, it is welcomed again
// if it comes back.
func (s *Service) ForgetWelcome(ctx context.Context, guildID string) error {
	return s.repo.ForgetWelcome(ctx, guildID)
}
