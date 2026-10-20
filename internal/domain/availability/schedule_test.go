package availability

import (
	"fmt"
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

func TestPastSlots(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	const week = WeekKey("2026-W42")   // Monday 12 October 2026
	evening, _ := HourlyLayout(20, 26) // 20h to 2h the next day
	wednesday2230 := time.Date(2026, 10, 14, 22, 30, 0, 0, paris)

	past := PastSlots(evening, week, paris, wednesday2230)
	for day := range 2 {
		for slot, p := range past[day] {
			if !p {
				t.Errorf("day %d slot %d is over on Wednesday evening", day, slot)
			}
		}
	}
	// Wednesday: 20h and 21h are over, 22h is going on, 23h to 1h are to come.
	if want := []bool{true, true, false, false, false, false}; fmt.Sprint(past[2]) != fmt.Sprint(want) {
		t.Errorf("Wednesday at 22h30: %v, want %v", past[2], want)
	}
	// Tuesday's 1h slot is on Wednesday at 1h: over.
	tuesdayNight := time.Date(2026, 10, 14, 1, 30, 0, 0, paris)
	if want := []bool{true, true, true, true, true, false}; fmt.Sprint(PastSlots(evening, week, paris, tuesdayNight)[1]) != fmt.Sprint(want) {
		t.Errorf("Tuesday's slots at Wednesday 1h30: %v, want %v", PastSlots(evening, week, paris, tuesdayNight)[1], want)
	}

	halfDay := PastSlots(HalfDayLayout(), week, paris, wednesday2230)
	if fmt.Sprint(halfDay[1]) != "[true true]" || fmt.Sprint(halfDay[2]) != "[false false]" {
		t.Errorf("half-day slots are over with their day: Tuesday %v, Wednesday %v", halfDay[1], halfDay[2])
	}
}

func TestClosedSlots(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	hourly, _ := HourlyLayout(8, 12) // 8h, 9h, 10h, 11h
	monday10 := DefaultSchedule()
	sunday18, _ := NewSchedule(time.Sunday, 18)
	const week = WeekKey("2026-W42")

	closed := ClosedSlots(hourly, monday10, week, paris)
	if want := []bool{true, true, false, false}; fmt.Sprint(closed[0]) != fmt.Sprint(want) {
		t.Errorf("Monday with a 10h publication: %v, want %v", closed[0], want)
	}
	if fmt.Sprint(closed[1]) != fmt.Sprint([]bool{false, false, false, false}) {
		t.Errorf("Tuesday is open: %v", closed[1])
	}
	for day, slots := range ClosedSlots(hourly, sunday18, week, paris) {
		for slot, c := range slots {
			if c {
				t.Errorf("published on Sunday, day %d slot %d is closed", day, slot)
			}
		}
	}
	for _, c := range ClosedSlots(HalfDayLayout(), monday10, week, paris)[0] {
		if c {
			t.Error("half-day slots have no hours: never closed")
		}
	}
}
