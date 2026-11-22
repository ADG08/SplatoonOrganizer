package discord

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// The /help panels are Components V2 messages: "Mes équipes" lists the guild's teams, or
// guides its setup when it has none, and each team has a panel listing its settings by
// section, each with its button: channel and role, publication (day and hour), slots,
// timezone (through its own picker) and message (display, language, green threshold).
const (
	// HelpPanelID starts the panels' custom IDs: <id>:<action>[:<teamID>[:<step>]].
	HelpPanelID = "help:config"

	HelpPanelColorDefault = 0x5865F2
	HelpPanelColorSuccess = 0x57F287
	HelpPanelColorDanger  = 0xED4245
)

// Actions of the panels; those of a team have its ID in the custom ID.
const (
	HelpActionList         = "list"
	HelpActionCreate       = "create"
	HelpActionModalCreate  = "modal_create"
	HelpActionOpen         = "open"
	HelpActionOpenPublish  = "open_publish"
	HelpActionOpenSchedule = "open_schedule"
	HelpActionOpenSlots    = "open_slots"
	HelpActionOpenTimezone = "open_timezone"
	HelpActionOpenMessage  = "open_message"
	HelpActionOpenRename   = "open_rename"
	HelpActionModalPublish = "modal_publish"
	HelpActionModalSched   = "modal_schedule"
	HelpActionModalSlots   = "modal_slots"
	HelpActionModalMessage = "modal_message"
	HelpActionModalRename  = "modal_rename"
	HelpActionTimezone     = "timezone" // the timezone picker's selects, with their step
	HelpActionPublishNow   = "publish_now"
	HelpActionDelete       = "delete"
	HelpActionConfirmDel   = "confirm_delete"

	// HelpActionLanguage, followed by a language, picks the guild's: "lang_en".
	HelpActionLanguage = "lang_"
)

// HelpCustomID builds a panel's custom ID; teamID 0 for an action without team.
func HelpCustomID(action string, teamID int64) string {
	id := HelpPanelID + ":" + action
	if teamID != 0 {
		id += ":" + strconv.FormatInt(teamID, 10)
	}
	return id
}

// ParseHelpCustomID reads a custom ID built by HelpCustomID, and the timezone picker's step
// that may follow it. The team's ID is 0 for an action without team.
func ParseHelpCustomID(customID string) (action string, teamID int64, step string, err error) {
	rest, ok := strings.CutPrefix(customID, HelpPanelID+":")
	if !ok {
		return "", 0, "", fmt.Errorf("custom id %q: no prefix %q", customID, HelpPanelID)
	}
	action, rest, hasTeam := strings.Cut(rest, ":")
	if action == "" {
		return "", 0, "", fmt.Errorf("custom id %q: no action", customID)
	}
	if !hasTeam {
		return action, 0, "", nil
	}
	id, step, _ := strings.Cut(rest, ":")
	if teamID, err = strconv.ParseInt(id, 10, 64); err != nil || teamID <= 0 {
		return "", 0, "", fmt.Errorf("custom id %q: invalid team id", customID)
	}
	return action, teamID, step, nil
}

// GuildLang returns the language of what concerns a whole guild (the list of its teams,
// the welcome message, its new teams): the one picked for it, else its oldest team's,
// else fallback.
func GuildLang(picked i18n.Lang, teams []*team.Team, fallback i18n.Lang) i18n.Lang {
	switch {
	case picked != "":
		return picked
	case len(teams) > 0:
		return teams[0].Language
	}
	return fallback
}

// languageFlags shows each language on its button.
var languageFlags = map[i18n.Lang]string{i18n.FR: "🇫🇷", i18n.EN: "🇬🇧"}

// languageButton switches the guild to the other language: "🇬🇧 EN" on French texts,
// "🇫🇷 FR" on English ones.
func languageButton(current i18n.Lang, customID func(i18n.Lang) string) discordgo.Button {
	other := i18n.Langs[0]
	for _, lang := range i18n.Langs {
		if lang != current {
			other = lang
			break
		}
	}
	return discordgo.Button{
		Label:    languageFlags[other] + " " + strings.ToUpper(string(other)),
		Style:    discordgo.SecondaryButton,
		CustomID: customID(other),
	}
}

// ProgressBar renders a setup's progress: "▰▱  1/2".
func ProgressBar(done, total int) string {
	done = max(0, min(done, total))
	return strings.Repeat("▰", done) + strings.Repeat("▱", total-done) + fmt.Sprintf("  %d/%d", done, total)
}

// checkLine is a step of the setup's checklist.
func checkLine(done bool, text string) string {
	if done {
		return "✅ " + text
	}
	return "⬜ " + text
}

// optionalLine is an optional step of the checklist, which the progress doesn't count.
func optionalLine(done bool, text string) string {
	if done {
		return "✅ " + text
	}
	return "▫️ " + text
}

// BuildTeamsPanel builds the first panel of /help: the setup when the guild has no team
// yet, where the guild's language is picked, else the list of its teams. status, when
// not empty, reports the last change; supportURL, when not empty, adds a link to support
// the bot.
func BuildTeamsPanel(m *i18n.Messages, teams []*team.Team, status, supportURL string) []discordgo.MessageComponent {
	var text, footer string
	createLabel := m.BtnCreateTeam
	if len(teams) == 0 {
		createLabel = m.BtnFirstTeam
		text = "## " + m.SetupTitle + "\n" + m.SetupIntro + "\n\n" + ProgressBar(0, team.SetupSteps) + "\n" +
			checkLine(false, m.StepTeam) + "\n" + checkLine(false, m.StepChannel) + "\n" + optionalLine(false, m.StepRole)
		footer = m.PanelFooter
	} else {
		lines := make([]string, len(teams))
		for n, t := range teams {
			lines[n] = teamLine(m, t)
		}
		text = "## " + m.TeamsTitle + "\n" + m.TeamsHint + "\n\n" + strings.Join(lines, "\n")
		footer = fmt.Sprintf(m.TeamsFooter, len(teams), team.MaxTeamsPerGuild)
	}
	if status != "" {
		text += "\n\n" + status
	}

	components := []discordgo.MessageComponent{discordgo.TextDisplay{Content: text}}
	components = append(components, teamButtons(teams)...)
	actions := []discordgo.MessageComponent{discordgo.Button{
		Label:    createLabel,
		Style:    discordgo.PrimaryButton,
		CustomID: HelpCustomID(HelpActionCreate, 0),
		Disabled: len(teams) >= team.MaxTeamsPerGuild,
	}}
	if len(teams) == 0 {
		actions = append(actions, languageButton(m.Lang(), func(l i18n.Lang) string {
			return HelpCustomID(HelpActionLanguage+string(l), 0)
		}))
	}
	if supportURL != "" {
		actions = append(actions, discordgo.Button{Label: m.BtnSupport, Style: discordgo.LinkButton, URL: supportURL})
	}
	components = append(components,
		discordgo.ActionsRow{Components: actions},
		discordgo.TextDisplay{Content: "-# " + footer},
	)

	accent := HelpPanelColorDefault
	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: components}}
}

// teamLine sums a team up in the list of the guild's teams.
func teamLine(m *i18n.Messages, t *team.Team) string {
	switch {
	case t.ChannelID != "" && !t.Active:
		return "⏸️ **" + t.Name + "** · <#" + t.ChannelID + "> · " + m.TeamPaused
	case t.IsComplete():
		line := "✅ **" + t.Name + "** · <#" + t.ChannelID + ">"
		if t.RoleID != "" {
			line += " · <@&" + t.RoleID + ">"
		}
		return line
	}
	return "🛠️ **" + t.Name + "** · " + ProgressBar(t.StepsDone(), team.SetupSteps)
}

// teamButtons opens each team's panel: green once set up, within Discord's 5 buttons per row.
func teamButtons(teams []*team.Team) []discordgo.MessageComponent {
	const buttonsPerRow = 5
	var rows []discordgo.MessageComponent
	for from := 0; from < len(teams); from += buttonsPerRow {
		var buttons []discordgo.MessageComponent
		for _, t := range teams[from:min(from+buttonsPerRow, len(teams))] {
			style := discordgo.SecondaryButton
			if t.IsComplete() {
				style = discordgo.SuccessButton
			}
			buttons = append(buttons, discordgo.Button{Label: t.Name, Style: style, CustomID: HelpCustomID(HelpActionOpen, t.ID)})
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons})
	}
	return rows
}

// BuildHelpPanel builds a team's panel, in its language: the progress of its setup, then
// one section per setting, its button on the right. status, when not empty, reports the
// last change under the title.
func BuildHelpPanel(t *team.Team, status string) []discordgo.MessageComponent {
	m := t.Messages()
	channel := m.NotConfigured
	if t.ChannelID != "" {
		channel = "<#" + t.ChannelID + ">"
		if !t.Active {
			channel = "⚠️ " + channel
		}
	}
	role := m.NoRole
	if t.RoleID != "" {
		role = "<@&" + t.RoleID + ">"
	}
	display := m.DisplayClassic
	if t.Display == team.DisplayImage {
		display = m.DisplayImage
	}
	fullAt := m.FullAtAuto
	if t.FullAt > 0 {
		fullAt = m.Players(t.FullAt)
	}

	header := "## 🦑 " + t.Name + "\n"
	accent := HelpPanelColorDefault
	switch {
	case t.IsComplete():
		header += fmt.Sprintf(m.TeamReady, t.Name)
		if t.RoleID == "" {
			header += "\n" + m.RoleTip
		}
		accent = HelpPanelColorSuccess
	default:
		header += ProgressBar(t.StepsDone(), team.SetupSteps) + "\n" + m.NextChannel
	}
	if status != "" {
		header += "\n\n" + status
	}
	section := func(title, action string, next bool, body ...string) discordgo.Section {
		style := discordgo.SecondaryButton
		if next {
			style = discordgo.PrimaryButton
		}
		return discordgo.Section{
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "### " + title + "\n" + strings.Join(body, "\n")},
			},
			Accessory: discordgo.Button{Label: m.BtnEdit, Style: style, CustomID: HelpCustomID(action, t.ID)},
		}
	}
	divider := discordgo.Separator{Divider: new(true), Spacing: new(discordgo.SeparatorSpacingSizeLarge)}

	components := []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: header},
		divider,
		section(m.SectionPublish, HelpActionOpenPublish, !t.IsComplete(),
			m.Field(m.FieldChannel, channel),
			m.Field(m.FieldRole, role),
		),
	}
	if t.ChannelID != "" && !t.Active {
		components = append(components, discordgo.TextDisplay{Content: "**" + m.PausedTitle + "**\n" + m.PausedHint})
	}
	components = append(components,
		section(m.SectionSchedule, HelpActionOpenSchedule, false, m.Field(m.FieldWhen, m.ScheduleSummary(t.Schedule))),
		divider,
		section(m.SectionSlots, HelpActionOpenSlots, false, slotsLines(t)...),
		section(m.SectionTimezone, HelpActionOpenTimezone, false,
			m.TimezoneName(t.Location)+" · "+m.Clock(time.Now().In(t.Location)),
		),
		divider,
		section(m.SectionMessage, HelpActionOpenMessage, false,
			m.Field(m.FieldDisplay, display),
			m.Field(m.FieldLanguage, m.LanguageName(t.Language)),
			m.Field(m.FieldFullAt, fullAt),
		),
		divider,
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: m.BtnBackToTeams, Style: discordgo.SecondaryButton, CustomID: HelpCustomID(HelpActionList, 0)},
			discordgo.Button{Label: m.BtnRename, Style: discordgo.SecondaryButton, CustomID: HelpCustomID(HelpActionOpenRename, t.ID)},
			discordgo.Button{Label: m.BtnPublishNow, Style: discordgo.SuccessButton, CustomID: HelpCustomID(HelpActionPublishNow, t.ID), Disabled: t.ChannelID == ""},
			discordgo.Button{Label: m.BtnDelete, Style: discordgo.DangerButton, CustomID: HelpCustomID(HelpActionDelete, t.ID)},
		}},
		discordgo.TextDisplay{Content: "-# " + m.PanelFooter},
	)

	return []discordgo.MessageComponent{discordgo.Container{AccentColor: &accent, Components: components}}
}

// BuildTeamDeletePanel asks to confirm the deletion of a team.
func BuildTeamDeletePanel(t *team.Team) []discordgo.MessageComponent {
	m := t.Messages()
	accent := HelpPanelColorDanger
	return []discordgo.MessageComponent{discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## " + fmt.Sprintf(m.DeleteTitle, t.Name) + "\n" + m.DeleteWarning},
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: m.BtnConfirmDelete, Style: discordgo.DangerButton, CustomID: HelpCustomID(HelpActionConfirmDel, t.ID)},
				discordgo.Button{Label: m.BtnCancel, Style: discordgo.SecondaryButton, CustomID: HelpCustomID(HelpActionOpen, t.ID)},
			}},
		},
	}}
}

// slotsLines describes the slots section: the layout and, for hourly slots, whose time
// the members see.
func slotsLines(t *team.Team) []string {
	m := t.Messages()
	lines := []string{m.LayoutSummary(t.SlotLayout)}
	if t.SlotLayout.IsHourly() {
		members := m.MembersServer
		if t.MemberTimezones {
			members = m.MembersOwn
		}
		lines = append(lines, m.Field(m.FieldMembers, members))
	}
	return lines
}
