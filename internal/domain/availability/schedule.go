package availability

import (
	"fmt"
	"time"
)

// DefaultPublishHour is the hour the weekly message is published at by default (Paris time).
const DefaultPublishHour = 10

// Schedule is when a guild's weekly message is published: on Sunday for the week that
// follows, or on Monday for the week that starts. Hour is in the bot's timezone (Paris).
type Schedule struct {
	Weekday time.Weekday // time.Sunday or time.Monday
	Hour    int          // 0-23
}

// DefaultSchedule returns the schedule of guilds that never chose one: Monday 10h.
func DefaultSchedule() Schedule {
	return Schedule{Weekday: time.Monday, Hour: DefaultPublishHour}
}

// NewSchedule returns a publication schedule, on Sunday or Monday.
func NewSchedule(weekday time.Weekday, hour int) (Schedule, error) {
	if weekday != time.Sunday && weekday != time.Monday {
		return Schedule{}, fmt.Errorf("invalid publication day %s: expected Sunday or Monday", weekday)
	}
	if hour < 0 || hour > 23 {
		return Schedule{}, fmt.Errorf("invalid publication hour %d: expected 0-23", hour)
	}
	return Schedule{Weekday: weekday, Hour: hour}, nil
}

// Week returns the week whose message is current at t: the latest one already published.
// Members' clicks go to that week, so they always match the message on display.
func (s Schedule) Week(t time.Time, loc *time.Location) WeekKey {
	t = t.In(loc)
	monday := time.Date(t.Year(), t.Month(), t.Day()-(int(t.Weekday())+6)%7, 0, 0, 0, 0, loc)
	for _, days := range []int{7, 0} {
		if start := monday.AddDate(0, 0, days); !t.Before(s.publishTime(start)) {
			return WeekKeyFromTime(start, loc)
		}
	}
	return WeekKeyFromTime(monday.AddDate(0, 0, -7), loc)
}

// publishTime returns when the message of the week starting on monday is published.
func (s Schedule) publishTime(monday time.Time) time.Time {
	day := monday.Day()
	if s.Weekday == time.Sunday {
		day--
	}
	// time.Date normalizes the day underflow and applies the DST offset of that exact day.
	return time.Date(monday.Year(), monday.Month(), day, s.Hour, 0, 0, 0, monday.Location())
}
