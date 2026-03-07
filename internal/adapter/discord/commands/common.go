package commands

import (
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

// ErrAdminOrOwner is the message shown when the user is neither admin nor bot owner.
const ErrAdminOrOwner = "Tu dois être administrateur du serveur ou le créateur du bot pour utiliser cette commande."

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
