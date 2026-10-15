package availability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// UserTimezone returns the timezone a member picked, or nil when they never did.
// Answers from memory after the first lookup.
func (s *Service) UserTimezone(ctx context.Context, userID string) (*time.Location, error) {
	if loc, ok := s.timezones.Load(userID); ok {
		return loc.(*time.Location), nil
	}
	zone, err := s.repo.UserTimezone(ctx, userID)
	if err != nil {
		return nil, err
	}
	var loc *time.Location
	if zone != "" {
		if loc, err = availability.ParseTimezone(zone); err != nil {
			// A zone removed from tzdata: ask the member again rather than failing.
			slog.WarnContext(ctx, "invalid stored timezone", "user", userID, "timezone", zone, "err", err)
			loc = nil
		}
	}
	s.timezones.Store(userID, loc)
	return loc, nil
}

// SetUserTimezone remembers the timezone a member picked.
func (s *Service) SetUserTimezone(ctx context.Context, userID string, loc *time.Location) error {
	if err := s.repo.SetUserTimezone(ctx, userID, loc.String()); err != nil {
		return err
	}
	s.timezones.Store(userID, loc)
	return nil
}

// SlotLabelIn labels a slot of the guild (timezone ref) in the member's timezone loc, e.g.
// "14h – 15h", prefixed with the local day when it differs from the guild's day ("Mar 3h – 4h"
// for Lundi 20h Paris seen from Tokyo). Half-day layouts and unknown timezones keep the
// guild's labels.
func (s *Service) SlotLabelIn(m *i18n.Messages, layout availability.Layout, ref *time.Location, week availability.WeekKey, dayIndex, slot int, loc *time.Location) string {
	if !layout.IsHourly() || loc == nil {
		return m.SlotLabel(layout, slot)
	}
	start, err := layout.SlotStart(week, dayIndex, slot, ref)
	if err != nil {
		return m.SlotLabel(layout, slot)
	}
	end, _ := layout.SlotStart(week, dayIndex, slot+1, ref)
	guildDay := start.Format("2006-01-02")
	start, end = start.In(loc), end.In(loc)

	endLabel := m.Clock(end)
	if end.Hour() == 0 && end.Minute() == 0 {
		endLabel = m.Midnight
	}
	label := m.Clock(start) + " – " + endLabel
	if start.Format("2006-01-02") != guildDay {
		label = m.ShortDays[weekdayIndex(start)] + " " + label
	}
	return label
}

// BuildWeeklyViewIn is BuildWeeklyView with the hours and days of the grid converted to
// the member's timezone, for the private "À mon heure" view.
func (s *Service) BuildWeeklyViewIn(m *i18n.Messages, layout availability.Layout, ref *time.Location, schedule availability.Schedule, week availability.WeekKey, counts []availability.SlotCount, loc *time.Location) WeeklyView {
	view := s.BuildWeeklyView(m, layout, ref, schedule, week, counts)
	if !layout.IsHourly() || loc == nil {
		return view
	}

	hours := make([]int, layout.Len())
	minutes := 0
	for slot := range hours {
		t, err := layout.SlotStart(week, 0, slot, ref)
		if err != nil {
			return view
		}
		t = t.In(loc)
		hours[slot], minutes = t.Hour(), t.Minute()
		view.Columns[slot] = m.Clock(t)
	}
	// A row keeps one evening together, labelled by the local day of its first slot
	// (an evening in Paris may start the next day in Asia or Oceania).
	for day := range view.Rows {
		if t, err := layout.SlotStart(week, day, 0, ref); err == nil {
			view.Rows[day] = m.ShortDays[weekdayIndex(t.In(loc))]
		}
	}

	view.Legend = fmt.Sprintf(m.LocalTimeLegend, m.TimezoneName(loc))
	if minutes != 0 {
		view.Legend += fmt.Sprintf(m.ShiftedColumns, minutes)
	}
	view.Legend += view.closedLegend(m)
	view.Table = hourlyTable(hours, view.Rows, view.Counts, view.Closed)
	return view
}

// weekdayIndex returns 0 for Monday ... 6 for Sunday.
func weekdayIndex(t time.Time) int {
	return (int(t.Weekday()) + 6) % 7
}
