package commands

import (
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// respondWithError sends an ephemeral error message to the user.
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// description returns a command text in French (the default) with its English
// localization: Discord shows command descriptions in each member's app language, as
// global commands can't follow a guild setting.
func description(text func(*i18n.Messages) string) (string, *map[discordgo.Locale]string) {
	en := text(i18n.For(i18n.EN))
	return text(i18n.For(i18n.FR)), &map[discordgo.Locale]string{discordgo.EnglishUS: en, discordgo.EnglishGB: en}
}

// isAdminOrOwner returns true if the user has Administrator permission or is the bot owner (DISCORD_OWNER_ID).
func isAdminOrOwner(i *discordgo.InteractionCreate, ownerID string) bool {
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	if ownerID != "" && i.Member.User != nil && i.Member.User.ID == ownerID {
		return true
	}
	return false
}
