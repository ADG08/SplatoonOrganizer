package availability

import (
	"strings"
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
	mustLoad(t, "America/Lima")
	for _, bad := range []string{"", "Mars/Base", "UTC+15"} {
		if _, err := availability.ParseTimezone(bad); err == nil {
			t.Errorf("ParseTimezone(%q): expected an error", bad)
		}
	}
}

func TestSlotLabelIn(t *testing.T) {
	paris := mustLoad(t, "Europe/Paris")
	svc := NewService(nil)
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
		{"Asia/Tokyo", 0, 7, "3h – 4h (lendemain)"}, // Lundi 20h Paris is Tuesday in Tokyo
		{"Asia/Kolkata", 0, 7, "23h30 – 0h30"},      // half-hour timezone
		{"America/Los_Angeles", 6, 0, "4h – 5h"},    // Sunday afternoon in Paris, early morning in LA
	}
	for _, c := range cases {
		if got := svc.SlotLabelIn(fr, layout, paris, week, c.day, c.slot, mustLoad(t, c.zone)); got != c.want {
			t.Errorf("%s day %d slot %d: got %q, want %q", c.zone, c.day, c.slot, got, c.want)
		}
	}
	if got := svc.SlotLabelIn(i18n.For(i18n.EN), layout, paris, week, 0, 7, mustLoad(t, "Asia/Tokyo")); got != "3 AM – 4 AM (next day)" {
		t.Errorf("English labels: got %q", got)
	}
	if got := svc.SlotLabelIn(fr, layout, paris, week, 0, 7, nil); got != "20h – 21h" {
		t.Errorf("unknown timezone keeps Paris labels: got %q", got)
	}
	if got := svc.SlotLabelIn(fr, availability.HalfDayLayout(), paris, week, 0, 1, mustLoad(t, "Asia/Tokyo")); got != "Soir" {
		t.Errorf("half-day labels are not converted: got %q", got)
	}

	// A guild in Montréal: its 20h is 2h the next day in Paris.
	montreal := mustLoad(t, "America/Toronto")
	if got := svc.SlotLabelIn(fr, layout, montreal, week, 0, 7, paris); got != "2h – 3h (lendemain)" {
		t.Errorf("Montréal guild seen from Paris: got %q", got)
	}
}

func TestBuildWeeklyViewIn(t *testing.T) {
	svc := NewService(nil)
	paris := mustLoad(t, "Europe/Paris")
	layout, _ := availability.HourlyLayout(13, 24)
	week := availability.WeekKey("2026-W42")

	tokyo := svc.BuildWeeklyViewIn(fr, layout, paris, availability.DefaultSchedule(), week, nil, mustLoad(t, "Asia/Tokyo"))
	if tokyo.Columns[0] != "20h" || tokyo.Columns[10] != "6h" || tokyo.Rows[0] != "Lun" {
		t.Errorf("Tokyo: columns %v, rows %v", tokyo.Columns, tokyo.Rows)
	}
	// 13h in Paris is already Tuesday midnight in New Zealand.
	auckland := svc.BuildWeeklyViewIn(fr, layout, paris, availability.DefaultSchedule(), week, nil, mustLoad(t, "Pacific/Auckland"))
	if auckland.Columns[0] != "0h" || auckland.Rows[0] != "Mar" || auckland.Rows[6] != "Lun" {
		t.Errorf("Auckland: columns %v, rows %v", auckland.Columns, auckland.Rows)
	}
	// The shared Paris view must not be affected by a member's view.
	if paris := svc.BuildWeeklyView(fr, layout, paris, availability.DefaultSchedule(), week, nil); paris.Rows[0] != "Lun" || paris.Columns[0] != "13h" {
		t.Errorf("Paris view changed: columns %v, rows %v", paris.Columns, paris.Rows)
	}
}

// An evening going on after midnight (15h -> 2h): its last slots are the next day's
// hours, shown in the row of the day the evening started.
func TestOvernightLayout(t *testing.T) {
	svc := NewService(nil)
	paris := mustLoad(t, "Europe/Paris")
	layout, err := availability.HourlyLayout(15, 26)
	if err != nil {
		t.Fatal(err)
	}
	const week = availability.WeekKey("2026-W42")

	if got := fr.LayoutSummary(layout); got != "Par heure · 15h → 2h (lendemain)" {
		t.Errorf("summary: %q", got)
	}
	if got := fr.SlotLabel(layout, 8) + " / " + fr.SlotLabel(layout, 9); got != "23h – minuit / 0h – 1h (lendemain)" {
		t.Errorf("labels after midnight: %q", got)
	}
	view := svc.BuildWeeklyView(fr, layout, paris, availability.DefaultSchedule(), week, nil)
	if got := view.Columns[8] + " " + view.Columns[9] + " " + view.Columns[10]; got != "23h 0h 1h" {
		t.Errorf("columns: %q", got)
	}
	if !strings.Contains(view.Table, "23  0  1") {
		t.Errorf("text table header should go 23, 0, 1:\n%s", view.Table)
	}
	// Monday's 1h slot is the night to Tuesday: marked as such in Paris, while in New York
	// it is still Monday 19h. In Los Angeles, Monday 15h Paris is still Monday morning, but
	// a slot early on Monday in Paris is Sunday there.
	if got := svc.SlotLabelIn(fr, layout, paris, week, 0, 10, paris); got != "1h – 2h (lendemain)" {
		t.Errorf("Monday 1h seen from Paris: %q", got)
	}
	if got := svc.SlotLabelIn(fr, layout, paris, week, 0, 10, mustLoad(t, "America/New_York")); got != "19h – 20h" {
		t.Errorf("Monday 1h seen from New York: %q", got)
	}
	early, _ := availability.HourlyLayout(6, 10)
	if got := svc.SlotLabelIn(fr, early, paris, week, 0, 0, mustLoad(t, "America/Los_Angeles")); got != "Dim 21h – 22h" {
		t.Errorf("Monday 6h seen from Los Angeles: %q", got)
	}
}
