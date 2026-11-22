package discord

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

var fr = i18n.For(i18n.FR)

// checkLimits checks a panel against Discord's limits: 40 components (nested ones
// included) and 4000 characters of text per message.
func checkLimits(t *testing.T, panel []discordgo.MessageComponent) {
	t.Helper()
	raw, err := json.Marshal(panel)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), `"type":`); n > 40 {
		t.Errorf("%d components, Discord allows 40", n)
	}
	total := 0
	for _, c := range panel[0].(discordgo.Container).Components {
		if text, ok := c.(discordgo.TextDisplay); ok {
			total += len([]rune(text.Content))
		}
	}
	if total > 4000 {
		t.Errorf("%d characters, Discord allows 4000", total)
	}
}

func TestHelpPanel(t *testing.T) {
	cfg := &team.Team{
		ID:         7,
		Name:       "Team A",
		ChannelID:  "c1",
		SlotLayout: availability.HalfDayLayout(),
		Schedule:   availability.DefaultSchedule(),
		Location:   availability.DefaultLocation(),
		Active:     false, // shows the paused notice too
	}
	panel := BuildHelpPanel(cfg, "✅ Configuration enregistrée.")
	checkLimits(t, panel)

	var buttons []string
	var text strings.Builder
	for _, c := range panel[0].(discordgo.Container).Components {
		switch c := c.(type) {
		case discordgo.TextDisplay:
			text.WriteString(c.Content + "\n")
		case discordgo.Section:
			text.WriteString(c.Components[0].(discordgo.TextDisplay).Content + "\n")
			buttons = append(buttons, c.Accessory.(discordgo.Button).CustomID)
		}
	}

	var want []string
	for _, action := range []string{HelpActionOpenPublish, HelpActionOpenSchedule, HelpActionOpenSlots, HelpActionOpenTimezone, HelpActionOpenMessage} {
		want = append(want, HelpCustomID(action, 7))
	}
	if strings.Join(buttons, ",") != strings.Join(want, ",") {
		t.Errorf("section buttons: %v, want %v", buttons, want)
	}
	for _, want := range []string{"✅ Configuration enregistrée.", "⚠️ <#c1>", "Publication en pause", "France", "est prête", fr.RoleTip} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("panel misses %q", want)
		}
	}
}

func TestHelpPanelSteps(t *testing.T) {
	base := team.Team{Name: "Team A", Active: true, SlotLayout: availability.HalfDayLayout(), Schedule: availability.DefaultSchedule(), Location: availability.DefaultLocation()}
	publishButton := func(panel []discordgo.MessageComponent) discordgo.Button {
		for _, c := range panel[0].(discordgo.Container).Components {
			if row, ok := c.(discordgo.ActionsRow); ok {
				return row.Components[2].(discordgo.Button)
			}
		}
		t.Fatal("no action row")
		return discordgo.Button{}
	}
	header := func(panel []discordgo.MessageComponent) string {
		return panel[0].(discordgo.Container).Components[0].(discordgo.TextDisplay).Content
	}

	empty := BuildHelpPanel(&base, "")
	if !strings.Contains(header(empty), fr.NextChannel) || !publishButton(empty).Disabled {
		t.Errorf("no channel: %q, publish %+v", header(empty), publishButton(empty))
	}

	// The channel is enough: the role is only suggested.
	ready := base
	ready.ChannelID = "c1"
	panel := BuildHelpPanel(&ready, "")
	if !strings.Contains(header(panel), "est prête") || !strings.Contains(header(panel), fr.RoleTip) || publishButton(panel).Disabled {
		t.Errorf("team with a channel: %q, publish %+v", header(panel), publishButton(panel))
	}
	ready.RoleID = "r1"
	panel = BuildHelpPanel(&ready, "")
	if !strings.Contains(header(panel), "est prête") || strings.Contains(header(panel), fr.RoleTip) {
		t.Errorf("team with a role: %q", header(panel))
	}
	if accent := *panel[0].(discordgo.Container).AccentColor; accent != HelpPanelColorSuccess {
		t.Errorf("complete team: accent %x", accent)
	}
}

func TestTeamsPanel(t *testing.T) {
	setup := BuildTeamsPanel(fr, nil, "", "https://revolut.me/example")
	checkLimits(t, setup)
	components := setup[0].(discordgo.Container).Components
	if text := components[0].(discordgo.TextDisplay).Content; !strings.Contains(text, "▱▱  0/2") || !strings.Contains(text, "▫️ "+fr.StepRole) {
		t.Errorf("the setup should show an empty progress bar: %q", text)
	}
	actions := components[1].(discordgo.ActionsRow).Components
	if len(actions) != 3 || actions[0].(discordgo.Button).Label != fr.BtnFirstTeam || actions[2].(discordgo.Button).URL != "https://revolut.me/example" {
		t.Fatalf("setup actions: %+v", actions)
	}
	// The language is picked from the start: one button switches to the other one.
	toEnglish := actions[1].(discordgo.Button)
	if toEnglish.Label != "🇬🇧 EN" || toEnglish.CustomID != HelpCustomID(HelpActionLanguage+"en", 0) {
		t.Errorf("language button: %+v", toEnglish)
	}
	if action, _, _, err := ParseHelpCustomID(toEnglish.CustomID); err != nil || action != HelpActionLanguage+"en" {
		t.Errorf("%q: %q, %v", toEnglish.CustomID, action, err)
	}
	inEnglish := BuildTeamsPanel(i18n.For(i18n.EN), nil, "", "")[0].(discordgo.Container).Components
	if text := inEnglish[0].(discordgo.TextDisplay).Content; !strings.Contains(text, i18n.For(i18n.EN).SetupTitle) {
		t.Errorf("setup in English: %q", text)
	}
	if toFrench := inEnglish[1].(discordgo.ActionsRow).Components[1].(discordgo.Button); toFrench.Label != "🇫🇷 FR" || toFrench.CustomID != HelpCustomID(HelpActionLanguage+"fr", 0) {
		t.Errorf("language button in English: %+v", toFrench)
	}

	teams := make([]*team.Team, team.MaxTeamsPerGuild)
	for n := range teams {
		teams[n] = &team.Team{
			ID: int64(n + 1), Name: strings.Repeat("é", team.MaxNameLength),
			ChannelID: "1566237124015956008", RoleID: "1566237244128366724", Active: true,
		}
	}
	full := BuildTeamsPanel(fr, teams, "⚠️ "+fr.TeamGone, "https://revolut.me/example")
	checkLimits(t, full)
	components = full[0].(discordgo.Container).Components
	var rows []discordgo.ActionsRow
	for _, c := range components {
		if row, ok := c.(discordgo.ActionsRow); ok {
			rows = append(rows, row)
		}
	}
	if len(rows) != 5 {
		t.Fatalf("%d rows, want 4 of teams and the create button", len(rows))
	}
	if !rows[4].Components[0].(discordgo.Button).Disabled {
		t.Error("the create button should be disabled at the limit of teams")
	}
	if len(rows[4].Components) != 2 {
		t.Errorf("with teams, only create and support: %+v", rows[4].Components)
	}
}

func TestTeamLine(t *testing.T) {
	for _, c := range []struct {
		team team.Team
		want string
	}{
		{team.Team{Name: "A", ChannelID: "c", RoleID: "r", Active: true}, "✅ **A** · <#c> · <@&r>"},
		{team.Team{Name: "B", ChannelID: "c", Active: true}, "✅ **B** · <#c>"}, // no role: optional
		{team.Team{Name: "C", Active: true}, "🛠️ **C** · ▰▱  1/2"},
		{team.Team{Name: "D", ChannelID: "c", RoleID: "r"}, "⏸️ **D** · <#c> · en pause"},
	} {
		if got := teamLine(fr, &c.team); got != c.want {
			t.Errorf("%+v: %q, want %q", c.team, got, c.want)
		}
	}
}

func TestHelpCustomID(t *testing.T) {
	for _, c := range []struct {
		customID string
		action   string
		teamID   int64
		step     string
	}{
		{HelpCustomID(HelpActionList, 0), HelpActionList, 0, ""},
		{HelpCustomID(HelpActionOpenPublish, 42), HelpActionOpenPublish, 42, ""},
		{HelpCustomID(HelpActionTimezone, 7) + ":w", HelpActionTimezone, 7, "w"},
	} {
		action, teamID, step, err := ParseHelpCustomID(c.customID)
		if err != nil || action != c.action || teamID != c.teamID || step != c.step {
			t.Errorf("%q: %q, %d, %q, %v", c.customID, action, teamID, step, err)
		}
	}
	for _, id := range []string{"help", "help:config", "help:config:", "other:open:1", "help:config:open:abc", "help:config:open:-1", "help:config:open:0"} {
		if _, _, _, err := ParseHelpCustomID(id); err == nil {
			t.Errorf("%q: expected an error", id)
		}
	}
}

func TestProgressBar(t *testing.T) {
	for _, c := range []struct {
		done int
		want string
	}{{0, "▱▱▱  0/3"}, {2, "▰▰▱  2/3"}, {3, "▰▰▰  3/3"}, {5, "▰▰▰  3/3"}} {
		if got := ProgressBar(c.done, 3); got != c.want {
			t.Errorf("ProgressBar(%d, 3) = %q, want %q", c.done, got, c.want)
		}
	}
}

func TestGuildLang(t *testing.T) {
	if got := GuildLang("", nil, i18n.EN); got != i18n.EN {
		t.Errorf("no team: %s", got)
	}
	teams := []*team.Team{{Language: i18n.FR}, {Language: i18n.EN}}
	if got := GuildLang("", teams, i18n.EN); got != i18n.FR {
		t.Errorf("oldest team's language: %s", got)
	}
	if got := GuildLang(i18n.EN, teams, i18n.FR); got != i18n.EN {
		t.Errorf("the picked language wins: %s", got)
	}
}

func TestWelcome(t *testing.T) {
	bot := &discordgo.User{ID: "42", Username: "Inkbot"}
	embed, components := BuildWelcome(fr, nil, bot)
	if embed.Title != "👋 Heeey, moi c'est Inkbot !" || embed.Thumbnail == nil || !strings.Contains(embed.Description, "▱▱  0/2") {
		t.Errorf("no team: %+v", embed)
	}
	buttons := components[0].(discordgo.ActionsRow).Components
	if len(components) != 1 || len(buttons) != 3 || buttons[0].(discordgo.Button).Label != fr.BtnStartSetup {
		t.Fatalf("buttons: %+v", components)
	}
	if toEnglish := buttons[2].(discordgo.Button); toEnglish.Label != "🇬🇧 EN" || toEnglish.CustomID != WelcomeLanguageCustomID+"en" {
		t.Errorf("language button: %+v", toEnglish)
	}

	en := i18n.For(i18n.EN)
	embed, components = BuildWelcome(en, nil, bot)
	toFrench := components[0].(discordgo.ActionsRow).Components[2].(discordgo.Button)
	if embed.Title != "👋 Heeey, I'm Inkbot!" || toFrench.Label != "🇫🇷 FR" || toFrench.CustomID != WelcomeLanguageCustomID+"fr" {
		t.Errorf("in English: %q, %+v", embed.Title, toFrench)
	}
	if embed, _ := BuildWelcome(fr, nil, nil); !strings.Contains(embed.Title, "SplatoonOrganizer") {
		t.Errorf("without the bot's user: %q", embed.Title)
	}

	// The checklist follows the most advanced team.
	// A channel completes the setup; the role, optional, only gets checked.
	embed, _ = BuildWelcome(fr, []*team.Team{{}, {ChannelID: "c"}}, bot)
	if !strings.Contains(embed.Description, "▰▰  2/2") || !strings.Contains(embed.Description, fr.WelcomeDone) || !strings.Contains(embed.Description, "▫️ "+fr.StepRole) {
		t.Errorf("channel set: %q", embed.Description)
	}

	embed, components = BuildWelcome(fr, []*team.Team{{ChannelID: "c", RoleID: "r"}}, bot)
	if !strings.Contains(embed.Description, fr.WelcomeDone) || !strings.Contains(embed.Description, "✅ "+fr.StepRole) || embed.Color != HelpPanelColorSuccess {
		t.Errorf("setup done: %+v", embed)
	}
	if label := components[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).Label; label != fr.BtnManageTeams {
		t.Errorf("setup done, button: %q", label)
	}
}
