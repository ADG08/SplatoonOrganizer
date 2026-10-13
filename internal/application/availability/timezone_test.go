package availability

import (
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

var fr = i18n.For(i18n.FR)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := availability.ParseTimezone(name)
	if err != nil {
		t.Fatalf("ParseTimezone(%q): %v", name, err)
	}
	return loc
}

func TestParseTimezone(t *testing.T) {
	for input, wantOffset := range map[string]int{
		"UTC+5:30": 5*3600 + 30*60,
		"gmt-3":    -3 * 3600,
		"+2":       2 * 3600,
	} {
		_, offset := time.Date(2026, 1, 1, 0, 0, 0, 0, mustLoad(t, input)).Zone()
		if offset != wantOffset {
			t.Errorf("%s: offset %d, want %d", input, offset, wantOffset)
		}
	}
	if got := mustLoad(t, "europe/paris").String(); got != "Europe/Paris" {
		t.Errorf("common zones are case-insensitive: got %q", got)
	}
	mustLoad(t, "America/Lima")
	for _, bad := range []string{"", "Mars/Base", "UTC+15"} {
		if _, err := availability.ParseTimezone(bad); err == nil {
			t.Errorf("ParseTimezone(%q): expected an error", bad)
		}
	}
}

func TestSlotLabelIn(t *testing.T) {
	paris := mustLoad(t, "Europe/Paris")
	svc := NewService(nil, paris)
	layout, _ := availability.HourlyLayout(13, 24)
	week := availability.WeekKey("2026-W42") // Monday 12/10/2026, summer time in Paris

	cases := []struct {
		zone      string
		day, slot int
		want      string
	}{
		{"Europe/Paris", 0, 7, "20h – 21h"},
		{"Europe/Paris", 0, 10, "23h – minuit"},
		{"America/New_York", 0, 7, "14h – 15h"},
		{"Asia/Tokyo", 0, 7, "Mar 3h – 4h"},      // Lundi 20h Paris is Tuesday in Tokyo
		{"Asia/Kolkata", 0, 7, "23h30 – 0h30"},   // half-hour timezone
		{"America/Los_Angeles", 6, 0, "4h – 5h"}, // Sunday afternoon in Paris, early morning in LA
	}
	for _, c := range cases {
		if got := svc.SlotLabelIn(fr, layout, week, c.day, c.slot, mustLoad(t, c.zone)); got != c.want {
			t.Errorf("%s day %d slot %d: got %q, want %q", c.zone, c.day, c.slot, got, c.want)
		}
	}
	if got := svc.SlotLabelIn(i18n.For(i18n.EN), layout, week, 0, 7, mustLoad(t, "Asia/Tokyo")); got != "Tue 3 AM – 4 AM" {
		t.Errorf("English labels: got %q", got)
	}
	if got := svc.SlotLabelIn(fr, layout, week, 0, 7, nil); got != "20h – 21h" {
		t.Errorf("unknown timezone keeps Paris labels: got %q", got)
	}
	if got := svc.SlotLabelIn(fr, availability.HalfDayLayout(), week, 0, 1, mustLoad(t, "Asia/Tokyo")); got != "Soir" {
		t.Errorf("half-day labels are not converted: got %q", got)
	}
}

func TestBuildWeeklyViewIn(t *testing.T) {
	svc := NewService(nil, mustLoad(t, "Europe/Paris"))
	layout, _ := availability.HourlyLayout(13, 24)
	week := availability.WeekKey("2026-W42")

	tokyo := svc.BuildWeeklyViewIn(fr, layout, week, nil, mustLoad(t, "Asia/Tokyo"))
	if tokyo.Columns[0] != "20h" || tokyo.Columns[10] != "6h" || tokyo.Rows[0] != "Lun" {
		t.Errorf("Tokyo: columns %v, rows %v", tokyo.Columns, tokyo.Rows)
	}
	// 13h in Paris is already Tuesday midnight in New Zealand.
	auckland := svc.BuildWeeklyViewIn(fr, layout, week, nil, mustLoad(t, "Pacific/Auckland"))
	if auckland.Columns[0] != "0h" || auckland.Rows[0] != "Mar" || auckland.Rows[6] != "Lun" {
		t.Errorf("Auckland: columns %v, rows %v", auckland.Columns, auckland.Rows)
	}
	// The shared Paris view must not be affected by a member's view.
	if paris := svc.BuildWeeklyView(fr, layout, week, nil); paris.Rows[0] != "Lun" || paris.Columns[0] != "13h" {
		t.Errorf("Paris view changed: columns %v, rows %v", paris.Columns, paris.Rows)
	}
}
