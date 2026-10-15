package guildconfig

import (
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

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
	SlotLayout availability.Layout   // how days are split into slots (half-day by default)
	Display    Display               // how the weekly message is shown (classic by default)
	Language   i18n.Lang             // language of the bot in the guild (French by default)
	Schedule   availability.Schedule // when the weekly message is published (Monday 10h by default)
	// Location is the timezone of the guild's hours: slots, grid, weeks and publication
	// (availability.DefaultTimezone by default).
	Location *time.Location
	// Active is false while the bot is out of the guild or can't post in its channel:
	// nothing is published, and the configuration is kept for when it comes back.
	Active bool
	// MemberTimezones is on for international guilds: each member is asked their
	// timezone and sees the hourly slots in it. Off, everyone uses Location.
	MemberTimezones bool
}

// MemberTime reports whether members see the slots in their own timezone: hourly slots
// (half-day ones have no hours) of a guild that turned MemberTimezones on.
func (c *GuildConfig) MemberTime() bool {
	return c.SlotLayout.IsHourly() && c.MemberTimezones
}

// Messages returns the texts in the guild's language.
func (c *GuildConfig) Messages() *i18n.Messages {
	return i18n.For(c.Language)
}
