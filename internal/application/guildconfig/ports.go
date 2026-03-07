package guildconfig

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
)

// Repository is the outbound port for guild configuration persistence.
type Repository interface {
	Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error)
	SetChannel(ctx context.Context, guildID, channelID string) error
	SetRole(ctx context.Context, guildID, roleID string) error
}
