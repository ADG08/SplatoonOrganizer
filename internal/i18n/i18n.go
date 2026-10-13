// Package i18n holds every text the bot shows to members, in each supported language,
// and the formatting that depends on the language (hours, days, slot labels).
package i18n

import (
	"fmt"
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

// SlotLabel returns the label of a slot in Paris time: "Soir" or "20h – 21h".
func (m *Messages) SlotLabel(l availability.Layout, slot int) string {
	if !l.IsHourly() {
		return m.HalfDay[slot]
	}
	start := l.StartHour + slot
	return m.HourLabel(start) + " – " + m.HourLabel(start+1)
}

// LayoutSummary describes a layout for the admin panel: "Par heure · 20h → minuit".
func (m *Messages) LayoutSummary(l availability.Layout) string {
	if !l.IsHourly() {
		return m.HalfDay[0] + " / " + m.HalfDay[1]
	}
	return fmt.Sprintf(m.HourlySummary, m.HourLabel(l.StartHour), m.HourLabel(l.EndHour))
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
