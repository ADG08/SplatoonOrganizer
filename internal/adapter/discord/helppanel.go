package discord

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	HelpPanelOpenChannelModalID = "help:config_open_channel_modal"
	HelpPanelOpenRoleModalID    = "help:config_open_role_modal"
	HelpPanelModalChannelID     = "help:config_modal_channel"
	HelpPanelModalRoleID        = "help:config_modal_role"

	HelpPanelColorDefault = 0x5865F2
	HelpPanelColorSuccess = 0x57F287
)

// BuildHelpPanelEmbed builds the configuration panel embed with the current channel/role state.
func BuildHelpPanelEmbed(title, description string, color int, channelID, roleID string) *discordgo.MessageEmbed {
	channelValue := "❌  Non configure"
	if channelID != "" {
		channelValue = "✅  <#" + channelID + ">"
	}
	roleValue := "❌  Non configure"
	if roleID != "" {
		roleValue = "✅  <@&" + roleID + ">"
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📢  Salon des dispos", Value: channelValue, Inline: true},
			{Name: "🏷️  Role a mentionner", Value: roleValue, Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "🔒 Reservé aux administrateurs"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// BuildHelpPanelComponents returns the action row with the channel and role config buttons.
func BuildHelpPanelComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "📢  Configurer le salon",
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenChannelModalID,
				},
				discordgo.Button{
					Label:    "🏷️  Configurer le role",
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenRoleModalID,
				},
			},
		},
	}
}
