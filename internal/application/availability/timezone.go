package availability

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// UserTimezone returns the timezone a member picked, or nil when they never did.
func (s *Service) UserTimezone(ctx context.Context, userID string) (*time.Location, error) {
	zone, err := s.repo.GetUserTimezone(ctx, userID)
	if err != nil || zone == "" {
		return nil, err
	}
	loc, err := availability.ParseTimezone(zone)
	if err != nil {
		// A zone removed from tzdata: ask the member again rather than failing.
		log.Printf("user %s: stored timezone %q: %v", userID, zone, err)
		return nil, nil
	}
	return loc, nil
}

// SetUserTimezone remembers the timezone a member picked.
func (s *Service) SetUserTimezone(ctx context.Context, userID string, loc *time.Location) error {
	return s.repo.SetUserTimezone(ctx, userID, loc.String())
}

// SlotLabelIn labels a slot in the member's timezone, e.g. "14h – 15h", prefixed with the
// local day when it differs from the reference day ("Mar 3h – 4h" for Lundi 20h Paris seen
// from Tokyo). Half-day layouts and unknown timezones keep the reference labels.
func (s *Service) SlotLabelIn(layout availability.Layout, week availability.WeekKey, dayIndex, slot int, loc *time.Location) string {
	if !layout.IsHourly() || loc == nil {
		return layout.Label(slot)
	}
	ref, err := layout.SlotStart(week, dayIndex, slot, s.loc)
	if err != nil {
		return layout.Label(slot)
	}
	end, _ := layout.SlotStart(week, dayIndex, slot+1, s.loc)
	start, end := ref.In(loc), end.In(loc)

	endLabel := availability.ClockLabel(end)
	if end.Hour() == 0 && end.Minute() == 0 {
		endLabel = "minuit"
	}
	label := availability.ClockLabel(start) + " – " + endLabel
	if start.Format("2006-01-02") != ref.Format("2006-01-02") {
		label = shortDayLabels[weekdayIndex(start)] + " " + label
	}
	return label
}

// BuildWeeklyViewIn is BuildWeeklyView with the hours and days of the grid converted to
// the member's timezone, for the private "À mon heure" view.
func (s *Service) BuildWeeklyViewIn(layout availability.Layout, week availability.WeekKey, counts []availability.SlotCount, loc *time.Location) WeeklyView {
	view := s.BuildWeeklyView(layout, week, counts)
	if !layout.IsHourly() || loc == nil {
		return view
	}

	hours := make([]int, layout.Len())
	minutes := 0
	for slot := range hours {
		t, err := layout.SlotStart(week, 0, slot, s.loc)
		if err != nil {
			return view
		}
		t = t.In(loc)
		hours[slot], minutes = t.Hour(), t.Minute()
		view.Columns[slot] = availability.ClockLabel(t)
	}
	// A row keeps one evening together, labelled by the local day of its first slot
	// (an evening in Paris may start the next day in Asia or Oceania).
	for day := range view.Rows {
		if t, err := layout.SlotStart(week, day, 0, s.loc); err == nil {
			view.Rows[day] = shortDayLabels[weekdayIndex(t.In(loc))]
		}
	}

	view.Legend = "Heures affichées à ton heure · " + availability.TimezoneName(loc)
	if minutes != 0 {
		view.Legend += fmt.Sprintf(" · colonnes décalées de %d min", minutes)
	}
	view.Table = hourlyTable(hours, view.Rows, view.Counts)
	return view
}

// weekdayIndex returns 0 for Monday ... 6 for Sunday.
func weekdayIndex(t time.Time) int {
	return (int(t.Weekday()) + 6) % 7
}
