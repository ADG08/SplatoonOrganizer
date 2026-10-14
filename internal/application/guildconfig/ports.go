package guildconfig

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Repository is the outbound port for guild configuration persistence.
type Repository interface {
	Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error)
	// ListWithChannel returns the guilds that configured a channel for the weekly message.
	ListWithChannel(ctx context.Context) ([]*guildconfig.GuildConfig, error)
	SetChannel(ctx context.Context, guildID, channelID string) error
	SetRole(ctx context.Context, guildID, roleID string) error
	SetMessageSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display, lang i18n.Lang) error
	SetSchedule(ctx context.Context, guildID string, schedule availability.Schedule) error
}
