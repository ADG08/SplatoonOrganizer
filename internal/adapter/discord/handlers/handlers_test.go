package handlers

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

var fr = i18n.For(i18n.FR)

func TestSplitLines(t *testing.T) {
	text := strings.Repeat("ligne é\n", 1000)
	chunks := splitLines(text, 100)
	if strings.Join(chunks, "") != text {
		t.Fatal("chunks don't add up to the text")
	}
	for _, c := range chunks {
		if len(c) > 100 || !utf8.ValidString(c) {
			t.Fatalf("chunk of %d bytes, valid UTF-8: %v", len(c), utf8.ValidString(c))
		}
	}
	// A line longer than the limit is cut on a rune boundary.
	for _, c := range splitLines(strings.Repeat("é", 100), 7) {
		if !utf8.ValidString(c) {
			t.Fatalf("invalid chunk %q", c)
		}
	}
	if got := splitLines("", 10); len(got) != 1 {
		t.Errorf("empty text: %q", got)
	}
}

func TestParseDayIndex(t *testing.T) {
	if day, err := parseDayIndex("6"); err != nil || day != 6 {
		t.Errorf("day 6: %d, %v", day, err)
	}
	for _, s := range []string{"", "7", "-1", "x"} {
		if _, err := parseDayIndex(s); err == nil {
			t.Errorf("%q: expected an error", s)
		}
	}
}

func TestUserDisposSelectMenus(t *testing.T) {
	layout, _ := availability.HourlyLayout(20, 24)
	label := func(day, slot int) string { return fr.SlotLabel(layout, slot) }
	rows := buildUserDisposSelectMenus(fr, 4, layout, map[int]map[int]bool{5: {1: true}}, nil, nil, 5, 2, label)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want Saturday and Sunday", len(rows))
	}
	menu := rows[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if menu.CustomID != SelectDisposCustomIDPrefix+":4:5" || menu.MaxValues != 4 || len(menu.Options) != 4 {
		t.Errorf("Saturday menu: %+v", menu)
	}
	// Values are 1-based slot indexes: the select handler and posted messages rely on it.
	if opt := menu.Options[1]; opt.Value != "2" || !opt.Default || opt.Label != "21h – 22h" {
		t.Errorf("Saturday 21h: %+v", opt)
	}
}

func TestClosedSlotsLeftOut(t *testing.T) {
	layout, _ := availability.HourlyLayout(8, 12) // 8h-12h, Monday 10h publication: 8h and 9h closed
	closed := availability.ClosedSlots(layout, availability.DefaultSchedule(), "2026-W42", availability.DefaultLocation())
	label := func(day, slot int) string { return fr.SlotLabel(layout, slot) }
	menu := buildUserDisposSelectMenus(fr, 4, layout, nil, closed, nil, 0, 1, label)[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.Options) != 2 || menu.Options[0].Value != "3" || menu.MaxValues != 2 {
		t.Errorf("Monday should offer 10h and 11h only, as values 3 and 4: %+v", menu.Options)
	}

	early, _ := availability.HourlyLayout(6, 9) // every Monday slot is before 10h
	closed = availability.ClosedSlots(early, availability.DefaultSchedule(), "2026-W42", availability.DefaultLocation())
	menu = buildUserDisposSelectMenus(fr, 4, early, nil, closed, nil, 0, 1, label)[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if !menu.Disabled || len(menu.Options) != 1 {
		t.Errorf("a day without open slot gets a disabled menu: %+v", menu)
	}
}

func TestTimezonePicker(t *testing.T) {
	rows := buildTimezonePicker(fr, weekly.CustomID(TimezoneCustomIDPrefix, 1, tzNextDispos))
	for _, row := range rows {
		menu := row.(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
		if len(menu.Options) > 25 {
			t.Errorf("%s: %d options, Discord allows 25", menu.CustomID, len(menu.Options))
		}
		for _, opt := range menu.Options {
			if utf8.RuneCountInString(opt.Description) > 100 {
				t.Errorf("%s: description of %d characters, Discord allows 100", opt.Value, utf8.RuneCountInString(opt.Description))
			}
			if _, err := availability.ParseTimezone(opt.Value); err != nil {
				t.Errorf("option %q: %v", opt.Value, err)
			}
		}
	}
}

func TestPublishModal(t *testing.T) {
	fields := buildPublishModalComponents(&team.Team{Name: "Team A", ChannelID: "c1"})
	menu := func(n int) discordgo.SelectMenu { return fields[n].(discordgo.Label).Component.(discordgo.SelectMenu) }
	input := func(n int) discordgo.TextInput { return fields[n].(discordgo.Label).Component.(discordgo.TextInput) }
	channel, role := menu(0), menu(2)
	if len(channel.DefaultValues) != 1 || channel.DefaultValues[0].ID != "c1" || *channel.MinValues != 0 {
		t.Errorf("the channel is prefilled, and can be left out for a typed name: %+v", channel)
	}
	if len(role.DefaultValues) != 0 || *role.MinValues != 0 {
		t.Errorf("the role is optional, empty without one: %+v", role)
	}
	if *input(1).Required || *input(3).Required {
		t.Error("the names to create are optional")
	}

	// What the form sends back is read by handlePublishSubmit.
	channel.Values = []string{"c2"}
	newRole := input(3)
	newRole.Value = "  Joueurs  "
	submitted := []discordgo.MessageComponent{discordgo.Label{Component: channel}, discordgo.Label{Component: newRole}}
	if got := modalSelectValue(submitted, "channel") + "/" + modalTextValue(submitted, "new_role"); got != "c2/Joueurs" {
		t.Errorf("submitted values: %s", got)
	}
	if got := modalTextValue(submitted, "new_channel"); got != "" {
		t.Errorf("no name typed: %q", got)
	}
}

func TestScheduleModal(t *testing.T) {
	schedule, _ := availability.NewSchedule(time.Sunday, 18)
	fields := buildScheduleModalComponents(&team.Team{Schedule: schedule})
	menu := func(n int) discordgo.SelectMenu { return fields[n].(discordgo.Label).Component.(discordgo.SelectMenu) }
	day, hour := menu(0), menu(1)
	if !day.Options[0].Default || day.Options[1].Default {
		t.Errorf("Sunday should be preselected: %+v", day.Options)
	}
	if len(hour.Options) != 24 || !hour.Options[18].Default {
		t.Errorf("18h should be preselected among 24 hours")
	}

	// What the form sends back is read by handleScheduleSubmit.
	day.Values, hour.Values = []string{day.Options[0].Value}, []string{hour.Options[18].Value}
	submitted := []discordgo.MessageComponent{discordgo.Label{Component: day}, discordgo.Label{Component: hour}}
	if got := modalSelectValue(submitted, "publish_weekday") + "/" + modalSelectValue(submitted, "publish_hour"); got != "0/18" {
		t.Errorf("submitted values: %s", got)
	}
}

func TestModalsFitDiscord(t *testing.T) {
	layout, _ := availability.HourlyLayout(13, 24)
	cfg := &team.Team{Name: strings.Repeat("é", team.MaxNameLength), SlotLayout: layout, Language: i18n.EN, Schedule: availability.DefaultSchedule()}
	for name, fields := range map[string][]discordgo.MessageComponent{
		"publish":  buildPublishModalComponents(cfg),
		"schedule": buildScheduleModalComponents(cfg),
		"slots":    buildSlotsModalComponents(cfg),
		"message":  buildMessageModalComponents(cfg),
		"name":     buildTeamNameField(i18n.For(i18n.EN), cfg.Name),
	} {
		if len(fields) > 5 {
			t.Errorf("%s: %d fields, Discord allows 5 per form", name, len(fields))
		}
		for _, f := range fields {
			label := f.(discordgo.Label)
			if utf8.RuneCountInString(label.Label) > 45 || utf8.RuneCountInString(label.Description) > 100 {
				t.Errorf("%s: label %q or its description is too long for Discord", name, label.Label)
			}
			if menu, ok := label.Component.(discordgo.SelectMenu); ok && len(menu.Options) > 25 {
				t.Errorf("%s: %s has %d options, Discord allows 25", name, menu.CustomID, len(menu.Options))
			}
			if input, ok := label.Component.(discordgo.TextInput); ok && utf8.RuneCountInString(input.Placeholder) > 100 {
				t.Errorf("%s: %s has a placeholder too long for Discord", name, input.CustomID)
			}
		}
	}
}

// Every panel's custom ID must fit Discord's limit, and be read back as it was built.
func TestTimezonePickerCustomIDs(t *testing.T) {
	idBase := weekly.CustomID(TimezoneCustomIDPrefix, 1<<62, tzNextDispos)
	rows := buildTimezonePicker(fr, idBase)
	id := rows[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu).CustomID
	if len(id) > 100 {
		t.Errorf("%q: Discord allows 100 characters", id)
	}
	teamID, rest, err := weekly.ParseCustomID(id, TimezoneCustomIDPrefix)
	if err != nil || teamID != 1<<62 || rest != tzNextDispos+":"+tzStepWest {
		t.Errorf("%q: %d, %q, %v", id, teamID, rest, err)
	}
}

func TestCountryPicker(t *testing.T) {
	summer := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for offset, places := range availability.CountriesByOffset(summer) {
		name := availability.FormatUTCOffset(offset)
		rows := buildCountryPicker(fr, weekly.CustomID(TimezoneCustomIDPrefix, 1, tzNextDispos), name, places)
		menu := rows[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
		if len(menu.Options) > 25 {
			t.Errorf("%s: %d options, Discord allows 25", name, len(menu.Options))
		}
		values := map[string]bool{}
		for _, opt := range menu.Options {
			if values[opt.Value] {
				t.Errorf("%s: duplicate value %q, Discord rejects the menu", name, opt.Value)
			}
			values[opt.Value] = true
			if _, err := availability.ParseTimezone(opt.Value); err != nil {
				t.Errorf("%s: option %q: %v", name, opt.Value, err)
			}
		}
		if last := menu.Options[len(menu.Options)-1]; last.Value != name {
			t.Errorf("%s: the last option should keep the fixed offset, got %q", name, last.Value)
		}
		if offset == 2*3600 && !values["Europe/Paris"] {
			t.Errorf("UTC+2 in summer should offer France")
		}
	}
}
