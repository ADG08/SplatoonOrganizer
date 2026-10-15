package discord

import (
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// The panel lists the guild's settings by section, each with its button: publication
// (channel, role, day and hour), slots, timezone (through its own picker) and message
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
)

// BuildHelpPanel builds the configuration panel, a Components V2 message (send it with
// discordgo.MessageFlagsIsComponentsV2): one section per setting, its button on the
// right. status, when not empty, reports the last change under the title; supportURL,
// when not empty, adds a link to support the bot.
func BuildHelpPanel(cfg *guildconfig.GuildConfig, status, supportURL string) []discordgo.MessageComponent {
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
	fullAt := m.FullAtAuto
	if cfg.FullAt > 0 {
		fullAt = m.Players(cfg.FullAt)
	}

	header := "## " + m.PanelTitle + "\n" + m.PanelDescription
	if status != "" {
		header += "\n\n✅ " + status
	}
	section := func(title, customID string, body ...string) discordgo.Section {
		return discordgo.Section{
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "### " + title + "\n" + strings.Join(body, "\n")},
			},
			Accessory: discordgo.Button{Label: m.BtnEdit, Style: discordgo.SecondaryButton, CustomID: customID},
		}
	}
	divider := discordgo.Separator{Divider: new(true), Spacing: new(discordgo.SeparatorSpacingSizeLarge)}

	components := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: header},
		divider,
		section(m.SectionPublish, HelpPanelOpenPublishID,
			m.Field(m.FieldChannel, channel),
			m.Field(m.FieldRole, role),
			m.Field(m.FieldWhen, m.ScheduleSummary(cfg.Schedule)),
		),
	}
	if cfg.ChannelID != "" && !cfg.Active {
		components = append(components, discordgo.TextDisplay{Content: "**" + m.PausedTitle + "**\n" + m.PausedHint})
	}
	components = append(components,
		divider,
		section(m.SectionSlots, HelpPanelOpenSlotsID, slotsLines(cfg)...),
		section(m.SectionTimezone, HelpPanelOpenTimezoneID,
			m.TimezoneName(cfg.Location)+" · "+m.Clock(time.Now().In(cfg.Location)),
		),
		divider,
		section(m.SectionMessage, HelpPanelOpenMessageID,
			m.Field(m.FieldDisplay, display),
			m.Field(m.FieldLanguage, m.LanguageName(cfg.Language)),
			m.Field(m.FieldFullAt, fullAt),
		),
		divider,
	)
	if supportURL != "" {
		components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: m.BtnSupport, Style: discordgo.LinkButton, URL: supportURL},
		}})
	}
	components = append(components, discordgo.TextDisplay{Content: "-# " + m.PanelFooter})

	accent := HelpPanelColorDefault
	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: components}}
}

// slotsLines describes the slots section: the layout and, for hourly slots, whose time
// the members see.
func slotsLines(cfg *guildconfig.GuildConfig) []string {
	m := cfg.Messages()
	lines := []string{m.LayoutSummary(cfg.SlotLayout)}
	if cfg.SlotLayout.IsHourly() {
		members := m.MembersServer
		if cfg.MemberTimezones {
			members = m.MembersOwn
		}
		lines = append(lines, m.Field(m.FieldMembers, members))
	}
	return lines
}
