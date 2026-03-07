package availability

import (
	"fmt"
	"time"
)

// Constants for availability slots.
const (
	DaysPerWeek = 7
	SlotsPerDay = 2
	TotalSlots  = DaysPerWeek * SlotsPerDay
)

// WeekKey is the ISO week identifier (e.g. "2025-W09").
type WeekKey string

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
