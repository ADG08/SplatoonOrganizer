package guild

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Repository is the outbound port for what concerns a guild as a whole.
type Repository interface {
	// Get returns the guild, empty when nothing was saved for it.
	Get(ctx context.Context, guildID string) (*guild.Guild, error)
	SetLanguage(ctx context.Context, guildID string, lang i18n.Lang) error
	SetWelcome(ctx context.Context, guildID string, welcome guild.Welcome) error
	// ForgetWelcome keeps the guild's language.
	ForgetWelcome(ctx context.Context, guildID string) error
}
