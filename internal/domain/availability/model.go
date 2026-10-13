package availability

import (
	"fmt"
	"time"
)

// DaysPerWeek is the number of days in an availability week (0 = lundi ... 6 = dimanche).
const DaysPerWeek = 7

// WeekKey is the ISO week identifier (e.g. "2025-W09").
type WeekKey string

// WeeklyMessage is the message a guild's availabilities are shown in, for one week.
type WeeklyMessage struct {
	Week      WeekKey
	ChannelID string // empty for messages migrated from single-guild versions: use the guild's channel
	MessageID string
}

// SlotCount represents the number of users available for a given day/slot.
type SlotCount struct {
	DayIndex  int
	SlotIndex int
	Count     int
}

// SlotUsers represents users available for a given day/slot.
type SlotUsers struct {
	DayIndex  int
	SlotIndex int
	UserIDs   []string
}

// WeekKeyFromTime returns the ISO week key for the given time in the given location.
func WeekKeyFromTime(t time.Time, loc *time.Location) WeekKey {
	t = t.In(loc)
	year, week := t.ISOWeek()
	return WeekKey(fmt.Sprintf("%04d-W%02d", year, week))
}

// CurrentWeekKey returns the current week key in the given location.
func CurrentWeekKey(loc *time.Location) WeekKey {
	return WeekKeyFromTime(time.Now(), loc)
}

// Monday returns midnight of the Monday that starts the ISO week, in loc.
func (w WeekKey) Monday(loc *time.Location) (time.Time, error) {
	var year, week int
	if _, err := fmt.Sscanf(string(w), "%04d-W%02d", &year, &week); err != nil {
		return time.Time{}, fmt.Errorf("invalid week key %q: %w", w, err)
	}
	// January 4th is always in ISO week 1.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, loc)
	offset := (int(jan4.Weekday()) + 6) % 7 // days since Monday
	return time.Date(year, time.January, 4-offset+(week-1)*7, 0, 0, 0, 0, loc), nil
}
