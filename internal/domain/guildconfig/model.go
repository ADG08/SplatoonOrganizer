package guildconfig

import "github.com/ADG08/SplatoonOrganizer/internal/domain/availability"

// Display selects how the weekly availability message is shown.
type Display string

const (
	// DisplayClassic is the original message: an embed with a text table.
	DisplayClassic Display = "classic"
	// DisplayImage draws the grid as a picture inside a Components V2 container.
	DisplayImage Display = "image"
)

// GuildConfig holds optional channel and role IDs for a guild.
// Empty strings mean use default channel / no role ping.
type GuildConfig struct {
	GuildID    string
	ChannelID  string
	RoleID     string
	SlotLayout availability.Layout // how days are split into slots (half-day by default)
	Display    Display             // how the weekly message is shown (classic by default)
}
