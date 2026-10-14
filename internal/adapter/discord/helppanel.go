package discord

import (
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const (
	HelpPanelOpenChannelModalID = "help:config_open_channel_modal"
	HelpPanelOpenRoleModalID    = "help:config_open_role_modal"
	HelpPanelOpenSlotsModalID   = "help:config_open_slots_modal"
	HelpPanelOpenPublishModalID = "help:config_open_publish_modal"
	HelpPanelModalChannelID     = "help:config_modal_channel"
	HelpPanelModalRoleID        = "help:config_modal_role"
	HelpPanelModalSlotsID       = "help:config_modal_slots"
	HelpPanelModalPublishID     = "help:config_modal_publish"

	HelpPanelColorDefault = 0x5865F2
	HelpPanelColorSuccess = 0x57F287
)

// BuildHelpPanelEmbed builds the configuration panel embed with the guild's current
// settings, in the guild's language.
func BuildHelpPanelEmbed(title, description string, color int, cfg *guildconfig.GuildConfig) *discordgo.MessageEmbed {
	m := cfg.Messages()
	display := m.DisplayClassic
	if cfg.Display == guildconfig.DisplayImage {
		display = m.DisplayImage
	}
	language := m.LanguageNames[0]
	if cfg.Language == i18n.EN {
		language = m.LanguageNames[1]
	}
	channelValue := m.NotConfigured
	if cfg.ChannelID != "" {
		channelValue = "✅  <#" + cfg.ChannelID + ">"
		if !cfg.Active {
			channelValue = "⚠️  <#" + cfg.ChannelID + ">\n" + m.ChannelPaused
		}
	}
	roleValue := m.NotConfigured
	if cfg.RoleID != "" {
		roleValue = "✅  <@&" + cfg.RoleID + ">"
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: m.PanelChannel, Value: channelValue, Inline: true},
			{Name: m.PanelRole, Value: roleValue, Inline: true},
			{Name: m.PanelMessage, Value: m.LayoutSummary(cfg.SlotLayout) + "\n" + display + "\n" + language, Inline: true},
			{Name: m.PanelPublication, Value: m.ScheduleSummary(cfg.Schedule), Inline: true},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: m.PanelFooter},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// BuildHelpPanelComponents returns the action row with the channel, role, message and
// publication config buttons.
func BuildHelpPanelComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    m.BtnConfigChannel,
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenChannelModalID,
				},
				discordgo.Button{
					Label:    m.BtnConfigRole,
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenRoleModalID,
				},
				discordgo.Button{
					Label:    m.BtnConfigMessage,
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenSlotsModalID,
				},
				discordgo.Button{
					Label:    m.BtnConfigPublication,
					Style:    discordgo.PrimaryButton,
					CustomID: HelpPanelOpenPublishModalID,
				},
			},
		},
	}
}
