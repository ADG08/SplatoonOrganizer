package discord

import (
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

const (
	HelpPanelOpenChannelModalID = "help:config_open_channel_modal"
	HelpPanelOpenRoleModalID    = "help:config_open_role_modal"
	HelpPanelOpenSlotsModalID   = "help:config_open_slots_modal"
	HelpPanelModalChannelID     = "help:config_modal_channel"
	HelpPanelModalRoleID        = "help:config_modal_role"
	HelpPanelModalSlotsID       = "help:config_modal_slots"

	HelpPanelColorDefault = 0x5865F2
	HelpPanelColorSuccess = 0x57F287
)

// BuildHelpPanelEmbed builds the configuration panel embed with the current channel/role/slots state.
func BuildHelpPanelEmbed(title, description string, color int, cfg *guildconfig.GuildConfig) *discordgo.MessageEmbed {
	channelID, roleID := cfg.ChannelID, cfg.RoleID
	display := "Classique (texte)"
	if cfg.Display == guildconfig.DisplayImage {
		display = "Image"
	}
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
			{Name: "🕒  Creneaux", Value: cfg.SlotLayout.Summary() + "\n" + display, Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: "🔒 Reservé aux administrateurs"},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// BuildHelpPanelComponents returns the action row with the channel, role and slots config buttons.
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
				discordgo.Button{
					Label:    "🕒  Configurer les creneaux",
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenSlotsModalID,
				},
			},
		},
	}
}
