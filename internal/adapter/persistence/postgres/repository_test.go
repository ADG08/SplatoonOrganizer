package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool returns a pool on a fresh schema of TEST_DATABASE_URL with every migration
// applied, dropped at the end of the test. Without TEST_DATABASE_URL, the test is skipped:
//
//	TEST_DATABASE_URL=postgres://splatoon:splatoon@localhost:5432/splatoon?sslmode=disable go test ./...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := t.Context()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ups, err := filepath.Glob("../../../../migrations/*.up.sql") // sorted, so in version order
	if err != nil || len(ups) == 0 {
		t.Fatalf("migrations not found: %v", err)
	}
	for _, up := range ups {
		sql, err := os.ReadFile(up)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", filepath.Base(up), err)
		}
	}
	return pool
}

func TestAvailabilityRepository(t *testing.T) {
	ctx := t.Context()
	repo := NewAvailabilityRepository(testPool(t))
	const week = availability.WeekKey("2026-W42")

	// Replacing a day's slots keeps the other days and the other guilds.
	must(t, repo.SetDayAvailability(ctx, "g1", week, "alice", 0, []int{0, 1, 2}))
	must(t, repo.SetDayAvailability(ctx, "g1", week, "alice", 1, []int{3}))
	must(t, repo.SetDayAvailability(ctx, "g1", week, "alice", 0, []int{1}))
	must(t, repo.SetDayAvailability(ctx, "g2", week, "alice", 0, []int{5}))
	got, err := repo.UserAvailability(ctx, "g1", week, "alice")
	must(t, err)
	if want := map[int]map[int]bool{0: {1: true}, 1: {3: true}}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("alice in g1: got %v, want %v", got, want)
	}

	// Picking a slot clears "unavailable for the week"; an empty day does not.
	must(t, repo.SetWeekUnavailable(ctx, "g1", week, "bob"))
	must(t, repo.SetDayAvailability(ctx, "g1", week, "bob", 2, nil))
	if unavailable, _ := repo.IsUserWeekUnavailable(ctx, "g1", week, "bob"); !unavailable {
		t.Error("an empty day cleared bob's unavailability")
	}
	must(t, repo.SetDayAvailability(ctx, "g1", week, "bob", 0, []int{1}))
	if unavailable, _ := repo.IsUserWeekUnavailable(ctx, "g1", week, "bob"); unavailable {
		t.Error("picking a slot kept bob unavailable")
	}

	counts, err := repo.AvailabilityCounts(ctx, "g1", week)
	must(t, err)
	if !slices.Contains(counts, availability.SlotCount{DayIndex: 0, SlotIndex: 1, Count: 2}) {
		t.Errorf("counts: %v, want 2 members on Monday slot 1", counts)
	}
	users, err := repo.AvailabilityUsers(ctx, "g1", week)
	must(t, err)
	for _, u := range users {
		if u.DayIndex == 0 && u.SlotIndex == 1 && !slices.Equal(u.UserIDs, []string{"alice", "bob"}) {
			t.Errorf("Monday slot 1: %v", u.UserIDs)
		}
	}

	must(t, repo.DeleteWeekAvailability(ctx, "g1", week))
	if got, _ := repo.UserAvailability(ctx, "g2", week, "alice"); len(got) != 1 {
		t.Errorf("resetting g1 touched g2: %v", got)
	}

	// Weekly message: none, then upserted.
	if msg, err := repo.WeeklyMessage(ctx, "g1"); err != nil || msg != nil {
		t.Errorf("no weekly message yet: %v, %v", msg, err)
	}
	must(t, repo.SetWeeklyMessage(ctx, "g1", availability.WeeklyMessage{Week: week, ChannelID: "c1", MessageID: "m1"}))
	must(t, repo.SetWeeklyMessage(ctx, "g1", availability.WeeklyMessage{Week: "2026-W43", ChannelID: "c1", MessageID: "m2"}))
	if msg, err := repo.WeeklyMessage(ctx, "g1"); err != nil || msg.MessageID != "m2" || msg.Week != "2026-W43" {
		t.Errorf("weekly message: %+v, %v", msg, err)
	}

	// Timezones are per member, across guilds.
	if zone, err := repo.UserTimezone(ctx, "alice"); err != nil || zone != "" {
		t.Errorf("no timezone yet: %q, %v", zone, err)
	}
	must(t, repo.SetUserTimezone(ctx, "alice", "UTC+9"))
	must(t, repo.SetUserTimezone(ctx, "alice", "UTC-5"))
	if zone, _ := repo.UserTimezone(ctx, "alice"); zone != "UTC-5" {
		t.Errorf("timezone: %q", zone)
	}
}

func TestGuildConfigRepository(t *testing.T) {
	ctx := t.Context()
	repo := NewGuildConfigRepository(testPool(t))

	cfg, err := repo.Get(ctx, "g1")
	must(t, err)
	want := guildconfig.GuildConfig{GuildID: "g1", SlotLayout: availability.HalfDayLayout(), Display: guildconfig.DisplayClassic, Language: i18n.FR, Schedule: availability.DefaultSchedule(), Active: true, Location: availability.DefaultLocation()}
	if *cfg != want {
		t.Errorf("defaults: got %+v, want %+v", *cfg, want)
	}

	layout, _ := availability.HourlyLayout(20, 24)
	schedule, _ := availability.NewSchedule(time.Sunday, 18)
	must(t, repo.SetChannel(ctx, "g1", "c1"))
	must(t, repo.SetRole(ctx, "g1", "r1"))
	must(t, repo.SetMessageSettings(ctx, "g1", layout, guildconfig.DisplayImage, i18n.EN))
	must(t, repo.SetSchedule(ctx, "g1", schedule))
	must(t, repo.SetRole(ctx, "g2", "r2")) // a guild without channel

	cfg, err = repo.Get(ctx, "g1")
	must(t, err)
	want = guildconfig.GuildConfig{GuildID: "g1", ChannelID: "c1", RoleID: "r1", SlotLayout: layout, Display: guildconfig.DisplayImage, Language: i18n.EN, Schedule: schedule, Active: true, Location: availability.DefaultLocation()}
	if *cfg != want {
		t.Errorf("saved: got %+v, want %+v", *cfg, want)
	}

	list, err := repo.ListWithChannel(ctx)
	must(t, err)
	if len(list) != 1 || *list[0] != want {
		t.Errorf("guilds with a channel: %v", list)
	}

	// A paused guild keeps its configuration and is not published until it comes back.
	must(t, repo.SetActive(ctx, "g1", false))
	must(t, repo.SetActive(ctx, "g3", false)) // no configuration: nothing to pause
	if list, _ := repo.ListWithChannel(ctx); len(list) != 0 {
		t.Errorf("paused guild still listed: %v", list)
	}
	cfg, err = repo.Get(ctx, "g1")
	must(t, err)
	if want.Active = false; *cfg != want {
		t.Errorf("paused: got %+v, want %+v", *cfg, want)
	}
	if cfg, _ := repo.Get(ctx, "g3"); !cfg.Active || cfg.Location.String() != "Europe/Paris" {
		t.Errorf("a guild without configuration is active, in Paris: %+v", cfg)
	}

	// Timezones: a country's (follows daylight saving time) or a fixed offset.
	for _, zone := range []string{"America/Toronto", "UTC+5:30"} {
		loc, err := availability.ParseTimezone(zone)
		must(t, err)
		must(t, repo.SetTimezone(ctx, "g2", loc))
		if cfg, err := repo.Get(ctx, "g2"); err != nil || cfg.Location.String() != zone || cfg.RoleID != "r2" {
			t.Errorf("timezone %s: got %v, %v", zone, cfg.Location, err)
		}
	}
	must(t, repo.SetFullAt(ctx, "g2", 4))
	if cfg, _ := repo.Get(ctx, "g2"); cfg.FullAt != 4 {
		t.Errorf("full at: %d, want 4", cfg.FullAt)
	}
	must(t, repo.SetFullAt(ctx, "g2", 0)) // back to relative
	if cfg, _ := repo.Get(ctx, "g2"); cfg.FullAt != 0 {
		t.Errorf("full at after reset: %d, want 0", cfg.FullAt)
	}
	must(t, repo.SetMemberTimezones(ctx, "g2", true))
	if cfg, _ := repo.Get(ctx, "g2"); !cfg.MemberTimezones {
		t.Error("member timezones not saved")
	}
	must(t, repo.SetChannel(ctx, "g1", "c2")) // setting a channel resumes it
	if list, _ := repo.ListWithChannel(ctx); len(list) != 1 || list[0].ChannelID != "c2" || !list[0].Active {
		t.Errorf("resumed guild: %v", list)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
