// Package i18n holds every text the bot shows to members, in each supported language,
// and the formatting that depends on the language (hours, days, slot labels).
package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// Lang is a language a guild can pick for the bot.
type Lang string

const (
	FR Lang = "fr"
	EN Lang = "en"
)

// Langs lists the supported languages, the default one first.
var Langs = []Lang{FR, EN}

// Parse returns the language stored for a guild, French when unknown or empty.
func Parse(s string) Lang {
	if Lang(s) == EN {
		return EN
	}
	return FR
}

// For returns the texts of a language.
func For(l Lang) *Messages {
	if l == EN {
		return &en
	}
	return &fr
}

// HourLabel formats an hour of the day, 24 being midnight at the end of the day:
// "20h" / "8 PM".
func (m *Messages) HourLabel(hour int) string {
	if hour == 24 {
		return m.Midnight
	}
	return m.Clock(time.Date(2000, 1, 1, hour, 0, 0, 0, time.UTC))
}

// Clock formats a time of day: "20h", "20h30" / "8 PM", "8:30 PM".
func (m *Messages) Clock(t time.Time) string {
	if m.lang == EN {
		if t.Minute() == 0 {
			return t.Format("3 PM")
		}
		return t.Format("3:04 PM")
	}
	if t.Minute() == 0 {
		return fmt.Sprintf("%dh", t.Hour())
	}
	return fmt.Sprintf("%dh%02d", t.Hour(), t.Minute())
}

// SlotLabel returns the label of a slot in the guild's time: "Soir", "20h – 21h", or
// "0h – 1h (lendemain)" for the hours after midnight of an evening going on.
func (m *Messages) SlotLabel(l availability.Layout, slot int) string {
	if !l.IsHourly() {
		return m.HalfDay[slot]
	}
	start := l.StartHour + slot
	label := m.HourLabel(start%24) + " – " + m.HourLabel(start+1)
	if start >= 24 {
		label += m.NextDay
	}
	return label
}

// UntilLabel labels the end of the last slot, marking the next day's hours:
// "minuit", "2h (lendemain)".
func (m *Messages) UntilLabel(hour int) string {
	if hour > 24 {
		return m.HourLabel(hour) + m.NextDay
	}
	return m.HourLabel(hour)
}

// LayoutSummary describes a layout for the admin panel: "Par heure · 20h → minuit".
func (m *Messages) LayoutSummary(l availability.Layout) string {
	if !l.IsHourly() {
		return m.HalfDay[0] + " / " + m.HalfDay[1]
	}
	return fmt.Sprintf(m.HourlySummary, m.HourLabel(l.StartHour), m.UntilLabel(l.EndHour))
}

// ScheduleSummary describes a publication schedule for the admin panel: "Lundi · 10h".
func (m *Messages) ScheduleSummary(s availability.Schedule) string {
	day := m.Days[0]
	if s.Weekday == time.Sunday {
		day = m.Days[6]
	}
	return day + " · " + m.HourLabel(s.Hour)
}

// Field formats a labelled value of the /help panel: "Salon : #dispos" / "Channel: #dispos".
func (m *Messages) Field(label, value string) string {
	if m.lang == EN {
		return label + ": " + value
	}
	return label + " : " + value
}

// LanguageName returns the name of a language, in that language: "Français", "English".
func (m *Messages) LanguageName(l Lang) string {
	for n, lang := range Langs {
		if lang == Parse(string(l)) {
			return m.LanguageNames[n]
		}
	}
	return m.LanguageNames[0]
}

// Players formats a number of players: "1 joueur", "4 joueurs".
func (m *Messages) Players(n int) string {
	if n == 1 {
		return m.PlayersOne
	}
	return fmt.Sprintf(m.PlayersMany, n)
}

// ColorLegend explains the grid image's colors. With slots full from fullAt players, it
// gives the actual thresholds ("🟩 4+ · 🟨 2–3 · ⬛ 1"); fullAt 0 keeps the relative
// "many · some · few" legend.
func (m *Messages) ColorLegend(fullAt int) string {
	if fullAt <= 0 {
		return m.ImageLegend
	}
	span := func(from, to int) string {
		if from == to {
			return strconv.Itoa(from)
		}
		return strconv.Itoa(from) + "–" + strconv.Itoa(to)
	}
	mid := availability.MidAt(fullAt)
	parts := []string{"🟩 " + strconv.Itoa(fullAt) + "+"}
	if mid < fullAt {
		parts = append(parts, "🟨 "+span(mid, fullAt-1))
	}
	if mid > 1 {
		parts = append(parts, "⬛ "+span(1, mid-1))
	}
	return strings.Join(parts, " · ")
}

// CountryName returns the name of a place of availability.Countries.
func (m *Messages) CountryName(tz availability.Timezone) string {
	if m.lang == EN {
		return tz.En
	}
	return tz.Fr
}

// TimezoneName describes a location for members: its country when known, else the
// IANA name or the UTC offset.
func (m *Messages) TimezoneName(loc *time.Location) string {
	for _, tz := range availability.Countries {
		if tz.Zone == loc.String() {
			return m.CountryName(tz)
		}
	}
	return loc.String()
}

// WeekTitle returns the title of the weekly message: "Dispos de la semaine du 12/10".
func (m *Messages) WeekTitle(monday time.Time) string {
	return fmt.Sprintf(m.WeeklyTitle, monday.Format(m.DateFormat))
}
