package availability

import (
	"testing"
	"time"
)

func TestScheduleWeek(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	at := func(date string) time.Time {
		tm, err := time.ParseInLocation("2006-01-02 15:04", date, paris)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}
	monday10 := DefaultSchedule()
	sunday18, _ := NewSchedule(time.Sunday, 18)
	sunday0, _ := NewSchedule(time.Sunday, 0)

	cases := []struct {
		schedule Schedule
		at       string
		want     WeekKey
	}{
		// 2026-W42 starts on Monday 2026-10-12.
		{monday10, "2026-10-12 09:59", "2026-W41"}, // last week's message is still on display
		{monday10, "2026-10-12 10:00", "2026-W42"},
		{monday10, "2026-10-18 23:59", "2026-W42"},
		{sunday18, "2026-10-11 17:59", "2026-W41"},
		{sunday18, "2026-10-11 18:00", "2026-W42"}, // Sunday evening: the week that follows
		{sunday18, "2026-10-14 12:00", "2026-W42"},
		{sunday18, "2026-10-18 18:00", "2026-W43"},
		{sunday0, "2026-10-25 00:00", "2026-W44"},  // day of the switch to winter time
		{sunday18, "2026-01-04 18:00", "2026-W02"}, // across the new year: ISO 2026-W01 starts on 2025-12-29
		{sunday18, "2025-12-28 18:00", "2026-W01"},
	}
	for _, c := range cases {
		if got := c.schedule.Week(at(c.at), paris); got != c.want {
			t.Errorf("%+v at %s: got %s, want %s", c.schedule, c.at, got, c.want)
		}
	}
}

func TestNewScheduleValidation(t *testing.T) {
	for _, c := range []struct {
		day  time.Weekday
		hour int
	}{{time.Tuesday, 10}, {time.Saturday, 10}, {time.Monday, -1}, {time.Sunday, 24}} {
		if _, err := NewSchedule(c.day, c.hour); err == nil {
			t.Errorf("NewSchedule(%s, %d): expected error", c.day, c.hour)
		}
	}
}
