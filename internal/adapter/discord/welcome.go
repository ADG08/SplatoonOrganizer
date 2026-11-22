package discord

import (
	"fmt"
	"strings"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// Custom IDs of the welcome message's buttons, stored in the messages already posted.
const (
	WelcomeCustomIDPrefix = "welcome"
	WelcomeSetupCustomID  = WelcomeCustomIDPrefix + ":setup"
	WelcomeHowToCustomID  = WelcomeCustomIDPrefix + ":howto"
	// WelcomeLanguageCustomID, followed by a language, picks the guild's: "welcome:lang:en".
	WelcomeLanguageCustomID = WelcomeCustomIDPrefix + ":lang:"
)

// BuildWelcome builds the public message the bot introduces itself with when it joins a
// guild, an embed with its buttons, in m's language: admins switch it to the other one
// there. bot gives the bot's name and avatar; the checklist follows the guild's most
// advanced team.
func BuildWelcome(m *i18n.Messages, teams []*team.Team, bot *discordgo.User) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	best, role := 0, false
	for _, t := range teams {
		best = max(best, t.StepsDone())
		role = role || t.RoleID != ""
	}
	lines := []string{
		m.WelcomeIntro,
		"",
		"**" + m.SetupTitle + "**   " + ProgressBar(best, team.SetupSteps),
		checkLine(best >= 1, m.StepTeam),
		checkLine(best >= 2, m.StepChannel),
		optionalLine(role, m.StepRole),
		"",
	}

	color, setupLabel := HelpPanelColorDefault, m.BtnStartSetup
	if best == team.SetupSteps {
		color, setupLabel = HelpPanelColorSuccess, m.BtnManageTeams
		lines = append(lines, m.WelcomeDone)
	} else {
		lines = append(lines, m.WelcomeStart)
	}

	name := "Squad Planner"
	if bot != nil && bot.Username != "" {
		name = bot.Username
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf(m.WelcomeTitle, name),
		Description: strings.Join(lines, "\n"),
		Color:       color,
		Footer:      &discordgo.MessageEmbedFooter{Text: m.WelcomeFooter},
	}
	if bot != nil {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: bot.AvatarURL("256")}
	}
	return embed, []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: setupLabel, Style: discordgo.PrimaryButton, CustomID: WelcomeSetupCustomID},
			discordgo.Button{Label: m.BtnHowTo, Style: discordgo.SecondaryButton, CustomID: WelcomeHowToCustomID},
			languageButton(m.Lang(), func(l i18n.Lang) string { return WelcomeLanguageCustomID + string(l) }),
		}},
	}
}
