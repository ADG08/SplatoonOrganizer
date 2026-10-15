package discord

import (
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// The panel has three sections, each opened by its button: publication (channel, role,
// day and hour), slots (layout, and the timezone through its own picker) and message
// (display and language).
const (
	HelpPanelOpenPublishID  = "help:config_open_publish"
	HelpPanelOpenSlotsID    = "help:config_open_slots"
	HelpPanelOpenTimezoneID = "help:config_open_timezone"
	HelpPanelOpenMessageID  = "help:config_open_message"
	HelpPanelModalPublishID = "help:config_modal_publish"
	HelpPanelModalSlotsID   = "help:config_modal_slots"
	HelpPanelModalMessageID = "help:config_modal_message"
	HelpPanelTimezoneID     = "help:config_timezone" // the timezone picker's selects: <id>:<step>

	HelpPanelColorDefault = 0x5865F2
	HelpPanelColorSuccess = 0x57F287
)

// BuildHelpPanelEmbed builds the configuration panel embed with the guild's current
// settings, in the guild's language.
func BuildHelpPanelEmbed(title, description string, color int, cfg *guildconfig.GuildConfig) *discordgo.MessageEmbed {
	m := cfg.Messages()
	channel := m.NotConfigured
	if cfg.ChannelID != "" {
		channel = "<#" + cfg.ChannelID + ">"
		if !cfg.Active {
			channel = "⚠️ " + channel
		}
	}
	role := m.NoRole
	if cfg.RoleID != "" {
		role = "<@&" + cfg.RoleID + ">"
	}
	display := m.DisplayClassic
	if cfg.Display == guildconfig.DisplayImage {
		display = m.DisplayImage
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: m.SectionPublish, Inline: true, Value: lines(
			m.Field(m.FieldChannel, channel),
			m.Field(m.FieldRole, role),
			m.Field(m.FieldWhen, m.ScheduleSummary(cfg.Schedule)),
		)},
		{Name: m.SectionSlots, Inline: true, Value: lines(
			m.LayoutSummary(cfg.SlotLayout),
			m.Field(m.FieldTimezone, m.TimezoneName(cfg.Location)+" · "+m.Clock(time.Now().In(cfg.Location))),
		)},
		{Name: m.SectionMessage, Inline: true, Value: lines(
			m.Field(m.FieldDisplay, display),
			m.Field(m.FieldLanguage, m.LanguageName(cfg.Language)),
		)},
	}
	if cfg.ChannelID != "" && !cfg.Active {
		fields = append(fields, &discordgo.MessageEmbedField{Name: m.PausedTitle, Value: m.PausedHint})
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{Text: m.PanelFooter},
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

func lines(l ...string) string {
	return strings.Join(l, "\n")
}

// BuildHelpPanelComponents returns the panel's buttons, in the order of its sections.
func BuildHelpPanelComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	button := func(label, customID string) discordgo.Button {
		return discordgo.Button{Label: label, Style: discordgo.PrimaryButton, CustomID: customID}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				button(m.BtnPublish, HelpPanelOpenPublishID),
				button(m.BtnSlots, HelpPanelOpenSlotsID),
				button(m.BtnTimezone, HelpPanelOpenTimezoneID),
				button(m.BtnMessage, HelpPanelOpenMessageID),
			},
		},
	}
}
