package i18n

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

var verb = regexp.MustCompile(`%[^%]`)

// TestCatalogsComplete checks every language fills every text, with the same format verbs.
func TestCatalogsComplete(t *testing.T) {
	ref := reflect.ValueOf(fr)
	for _, l := range Langs {
		m := reflect.ValueOf(*For(l))
		for i := 0; i < m.NumField(); i++ {
			field := m.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			texts, refs := stringFields(m.Field(i)), stringFields(ref.Field(i))
			for j, text := range texts {
				if text == "" {
					t.Errorf("%s: %s[%d] is empty", l, field.Name, j)
				}
				if got, want := verb.FindAllString(text, -1), verb.FindAllString(refs[j], -1); !reflect.DeepEqual(got, want) {
					t.Errorf("%s: %s has verbs %v, French has %v", l, field.Name, got, want)
				}
			}
		}
	}
}

func stringFields(v reflect.Value) []string {
	if v.Kind() == reflect.String {
		return []string{v.String()}
	}
	out := make([]string, v.Len())
	for i := range out {
		out[i] = v.Index(i).String()
	}
	return out
}

func TestHourFormats(t *testing.T) {
	hourly, _ := availability.HourlyLayout(20, 24)
	cases := []struct {
		lang Lang
		got  string
		want string
	}{
		{FR, For(FR).SlotLabel(hourly, 0), "20h – 21h"},
		{FR, For(FR).SlotLabel(hourly, 3), "23h – minuit"},
		{EN, For(EN).SlotLabel(hourly, 0), "8 PM – 9 PM"},
		{EN, For(EN).SlotLabel(hourly, 3), "11 PM – midnight"},
		{EN, For(EN).HourLabel(0), "12 AM"},
		{FR, For(FR).Clock(time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC)), "23h30"},
		{EN, For(EN).Clock(time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC)), "11:30 PM"},
		{FR, For(FR).SlotLabel(availability.HalfDayLayout(), 1), "Soir"},
		{EN, For(EN).LayoutSummary(hourly), "Hourly · 8 PM → midnight"},
		{FR, For(FR).LayoutSummary(availability.HalfDayLayout()), "Après-midi / Soir"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.lang, c.got, c.want)
		}
	}
	if Parse("") != FR || Parse("en") != EN || Parse("de") != FR {
		t.Error("Parse must default to French")
	}
}

func TestColorLegend(t *testing.T) {
	fr := For(FR)
	for fullAt, want := range map[int]string{
		0: fr.ImageLegend, // relative to the week's best slot
		1: "🟩 1+",
		2: "🟩 2+ · 🟨 1",
		4: "🟩 4+ · 🟨 2–3 · ⬛ 1",
		6: "🟩 6+ · 🟨 3–5 · ⬛ 1–2",
	} {
		if got := fr.ColorLegend(fullAt); got != want {
			t.Errorf("ColorLegend(%d) = %q, want %q", fullAt, got, want)
		}
	}
	if got := fr.Players(1) + ", " + For(EN).Players(4); got != "1 joueur, 4 players" {
		t.Errorf("Players: %q", got)
	}
}
