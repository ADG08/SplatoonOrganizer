package availability

import (
	"fmt"
	"time"
)

// SlotMode selects how a day is split into availability slots.
type SlotMode string

const (
	// ModeHalfDay splits a day into "Après-midi" and "Soir".
	ModeHalfDay SlotMode = "halfday"
	// ModeHourly splits a day into one-hour slots between StartHour and EndHour.
	ModeHourly SlotMode = "hourly"

	// MaxSlotsPerDay is bounded by Discord: a select menu holds at most 25 options.
	MaxSlotsPerDay = 25

	// DefaultStartHour and DefaultEndHour are the hourly range proposed by default:
	// 20h to minuit, Paris time.
	DefaultStartHour = 20
	DefaultEndHour   = 24
)

// Layout describes the slots of a day. Slot indexes go from 0 to Len()-1.
type Layout struct {
	Mode      SlotMode
	StartHour int // hourly only: first slot starts at StartHour:00
	EndHour   int // hourly only: last slot ends at EndHour:00 (24 = minuit)
}

// HalfDayLayout returns the "Après-midi / Soir" layout.
func HalfDayLayout() Layout {
	return Layout{Mode: ModeHalfDay}
}

// HourlyLayout returns a layout with one slot per hour in [startHour, endHour).
func HourlyLayout(startHour, endHour int) (Layout, error) {
	if startHour < 0 || endHour > 24 || startHour >= endHour {
		return Layout{}, fmt.Errorf("invalid hourly range %dh-%dh: expected 0 <= start < end <= 24", startHour, endHour)
	}
	if endHour-startHour > MaxSlotsPerDay {
		return Layout{}, fmt.Errorf("invalid hourly range %dh-%dh: at most %d slots per day", startHour, endHour, MaxSlotsPerDay)
	}
	return Layout{Mode: ModeHourly, StartHour: startHour, EndHour: endHour}, nil
}

// IsHourly reports whether slots are real hours (and can be shown as Discord timestamps).
func (l Layout) IsHourly() bool {
	return l.Mode == ModeHourly
}

// Len returns the number of slots per day.
func (l Layout) Len() int {
	if l.IsHourly() {
		return l.EndHour - l.StartHour
	}
	return 2
}

// Valid reports whether slot is a valid slot index for this layout.
func (l Layout) Valid(slot int) bool {
	return slot >= 0 && slot < l.Len()
}

// SlotStart returns the start time of a slot of the given week, in loc. Hourly layouts only.
func (l Layout) SlotStart(week WeekKey, dayIndex, slot int, loc *time.Location) (time.Time, error) {
	monday, err := week.Monday(loc)
	if err != nil {
		return time.Time{}, err
	}
	// time.Date normalizes the day overflow and applies the DST offset of that exact day.
	return time.Date(monday.Year(), monday.Month(), monday.Day()+dayIndex, l.StartHour+slot, 0, 0, 0, loc), nil
}
