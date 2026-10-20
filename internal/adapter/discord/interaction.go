package discord

import "github.com/bwmarrin/discordgo"

// RespondError answers an interaction with a private error message.
func RespondError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// UserName returns the name of the member or user behind an interaction, for the logs.
func UserName(i *discordgo.InteractionCreate) string {
	user := i.User
	if i.Member != nil && i.Member.User != nil {
		user = i.Member.User
	}
	if user == nil {
		return ""
	}
	return user.Username
}

// IsAdminOrOwner reports whether the member may configure the bot: a server administrator,
// or the bot's creator (ownerID, optional).
func IsAdminOrOwner(i *discordgo.InteractionCreate, ownerID string) bool {
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	return ownerID != "" && i.Member.User != nil && i.Member.User.ID == ownerID
}
