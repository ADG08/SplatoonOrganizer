package guild

import "github.com/ADG08/SplatoonOrganizer/internal/i18n"

// Guild is what concerns a guild as a whole, beside its teams.
type Guild struct {
	// Language is the one picked for what isn't a team's: the list of its teams, the
	// welcome message, its new teams. Empty until an admin picks one.
	Language i18n.Lang
	// Welcome is the message posted when the bot joined the guild, nil when there is none.
	Welcome *Welcome
}

// Welcome is the message posted when the bot joins a guild.
type Welcome struct {
	ChannelID string
	MessageID string
}
