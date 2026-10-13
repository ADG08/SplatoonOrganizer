package availability

import (
	"testing"
	"time"
)

func TestWeekKeyMonday(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	cases := map[WeekKey]string{
		"2026-W01": "2025-12-29", // ISO week 1 starts in the previous year
		"2026-W42": "2026-10-12",
		"2020-W53": "2020-12-28",
	}
	for week, want := range cases {
		got, err := week.Monday(paris)
		if err != nil {
			t.Fatalf("%s: %v", week, err)
		}
		if got.Format("2006-01-02") != want || got.Weekday() != time.Monday {
			t.Errorf("%s: got %s (%s), want %s", week, got.Format("2006-01-02"), got.Weekday(), want)
		}
		if WeekKeyFromTime(got, paris) != week {
			t.Errorf("%s: round trip gave %s", week, WeekKeyFromTime(got, paris))
		}
	}
}

func TestHourlySlotStart(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	layout, err := HourlyLayout(13, 24)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Len() != 11 || layout.Label(10) != "23h – minuit" {
		t.Fatalf("unexpected layout: len=%d last=%q", layout.Len(), layout.Label(10))
	}

	// 2026-W43: Europe switches to winter time on Sunday 2026-10-25.
	week := WeekKey("2026-W43")
	sat, _ := layout.SlotStart(week, 5, 7, paris) // Saturday 20h CEST = 18:00 UTC
	sun, _ := layout.SlotStart(week, 6, 7, paris) // Sunday 20h CET = 19:00 UTC
	if got := sat.UTC().Format("2006-01-02 15:04"); got != "2026-10-24 18:00" {
		t.Errorf("saturday 20h: got %s UTC", got)
	}
	if got := sun.UTC().Format("2006-01-02 15:04"); got != "2026-10-25 19:00" {
		t.Errorf("sunday 20h: got %s UTC", got)
	}

	// End of the last slot is minuit, i.e. the next day at 00:00.
	end, _ := layout.SlotStart(week, 6, layout.Len(), paris)
	if got := end.Format("2006-01-02 15:04"); got != "2026-10-26 00:00" {
		t.Errorf("end of sunday: got %s", got)
	}
}

func TestHourlyLayoutValidation(t *testing.T) {
	for _, r := range [][2]int{{13, 13}, {-1, 10}, {10, 25}, {20, 18}} {
		if _, err := HourlyLayout(r[0], r[1]); err == nil {
			t.Errorf("HourlyLayout(%d, %d): expected error", r[0], r[1])
		}
	}
	if _, err := HourlyLayout(0, 24); err != nil {
		t.Errorf("HourlyLayout(0, 24): %v", err)
	}
}

func TestCountriesTimezonesExist(t *testing.T) {
	for _, c := range Countries {
		if _, err := time.LoadLocation(c.Zone); err != nil {
			t.Errorf("%s (%s): %v", c.Where, c.Zone, err)
		}
	}
}

func TestCountriesAtFollowDaylightSaving(t *testing.T) {
	summer := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	winter := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	if got := CountriesByOffset(summer)[2*3600]; len(got) == 0 || got[0] != "France" {
		t.Errorf("summer UTC+2: %v", got)
	}
	if got := CountriesByOffset(winter)[3600]; len(got) == 0 || got[0] != "France" {
		t.Errorf("winter UTC+1: %v", got)
	}
}
