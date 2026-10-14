package handlers

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
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

func TestParseSelectDisposCustomID(t *testing.T) {
	if day, err := parseSelectDisposCustomID(SelectDisposCustomIDPrefix + ":6"); err != nil || day != 6 {
		t.Errorf("day 6: %d, %v", day, err)
	}
	for _, id := range []string{SelectDisposCustomIDPrefix + ":7", SelectDisposCustomIDPrefix + ":x", "availability:open:1"} {
		if _, err := parseSelectDisposCustomID(id); err == nil {
			t.Errorf("%q: expected an error", id)
		}
	}
}

func TestUserDisposSelectMenus(t *testing.T) {
	layout, _ := availability.HourlyLayout(20, 24)
	label := func(day, slot int) string { return fr.SlotLabel(layout, slot) }
	rows := buildUserDisposSelectMenus(fr, layout, map[int]map[int]bool{5: {1: true}}, 5, 2, label)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want Saturday and Sunday", len(rows))
	}
	menu := rows[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if menu.CustomID != SelectDisposCustomIDPrefix+":5" || menu.MaxValues != 4 || len(menu.Options) != 4 {
		t.Errorf("Saturday menu: %+v", menu)
	}
	// Values are 1-based slot indexes: the select handler and posted messages rely on it.
	if opt := menu.Options[1]; opt.Value != "2" || !opt.Default || opt.Label != "21h – 22h" {
		t.Errorf("Saturday 21h: %+v", opt)
	}
}

func TestTimezonePicker(t *testing.T) {
	rows := buildTimezonePicker(fr, tzNextDispos)
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
	schedule, _ := availability.NewSchedule(time.Sunday, 18)
	fields := buildPublishModalComponents(&guildconfig.GuildConfig{Schedule: schedule})
	day := fields[0].(discordgo.Label).Component.(discordgo.SelectMenu)
	hour := fields[1].(discordgo.Label).Component.(discordgo.SelectMenu)
	if !day.Options[0].Default || day.Options[1].Default {
		t.Errorf("Sunday should be preselected: %+v", day.Options)
	}
	if len(hour.Options) != 24 || !hour.Options[18].Default {
		t.Errorf("18h should be preselected among 24 hours")
	}

	// What the form sends back is read by handlePublishSubmit.
	day.Values, hour.Values = []string{day.Options[0].Value}, []string{hour.Options[18].Value}
	submitted := []discordgo.MessageComponent{discordgo.Label{Component: day}, discordgo.Label{Component: hour}}
	if got := modalSelectValue(submitted, "publish_weekday") + "/" + modalSelectValue(submitted, "publish_hour"); got != "0/18" {
		t.Errorf("submitted values: %s", got)
	}
}

func TestSlotsModalFitsDiscord(t *testing.T) {
	layout, _ := availability.HourlyLayout(13, 24)
	fields := buildSlotsModalComponents(&guildconfig.GuildConfig{SlotLayout: layout, Language: i18n.EN})
	if len(fields) > 5 {
		t.Errorf("%d fields, Discord allows 5 per form", len(fields))
	}
	for _, f := range fields {
		if menu := f.(discordgo.Label).Component.(discordgo.SelectMenu); len(menu.Options) > 25 {
			t.Errorf("%s: %d options, Discord allows 25", menu.CustomID, len(menu.Options))
		}
	}
}

func TestCountryPicker(t *testing.T) {
	summer := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	for offset, places := range availability.CountriesByOffset(summer) {
		name := availability.FormatUTCOffset(offset)
		rows := buildCountryPicker(fr, tzNextDispos, name, places)
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
