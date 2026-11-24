package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
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

// newTeams creates two teams of different guilds, for the rows that need one.
func newTeams(t *testing.T, pool *pgxpool.Pool) (int64, int64) {
	t.Helper()
	repo := NewTeamRepository(pool)
	a, err := repo.Create(t.Context(), "g1", "A", i18n.FR)
	must(t, err)
	b, err := repo.Create(t.Context(), "g2", "B", i18n.FR)
	must(t, err)
	return a.ID, b.ID
}

func TestAvailabilityRepository(t *testing.T) {
	ctx := t.Context()
	pool := testPool(t)
	repo := NewAvailabilityRepository(pool)
	t1, t2 := newTeams(t, pool)
	const week = availability.WeekKey("2026-W42")

	// Replacing a day's slots keeps the other days and the other teams.
	must(t, repo.SetDayAvailability(ctx, t1, week, "alice", 0, []int{0, 1, 2}))
	must(t, repo.SetDayAvailability(ctx, t1, week, "alice", 1, []int{3}))
	must(t, repo.SetDayAvailability(ctx, t1, week, "alice", 0, []int{1}))
	must(t, repo.SetDayAvailability(ctx, t2, week, "alice", 0, []int{5}))
	got, err := repo.UserAvailability(ctx, t1, week, "alice")
	must(t, err)
	if want := map[int]map[int]bool{0: {1: true}, 1: {3: true}}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("alice in team 1: got %v, want %v", got, want)
	}

	// Picking a slot clears "unavailable for the week"; an empty day does not.
	must(t, repo.SetWeekUnavailable(ctx, t1, week, "bob"))
	must(t, repo.SetDayAvailability(ctx, t1, week, "bob", 2, nil))
	if unavailable, _ := repo.IsUserWeekUnavailable(ctx, t1, week, "bob"); !unavailable {
		t.Error("an empty day cleared bob's unavailability")
	}
	must(t, repo.SetDayAvailability(ctx, t1, week, "bob", 0, []int{1}))
	if unavailable, _ := repo.IsUserWeekUnavailable(ctx, t1, week, "bob"); unavailable {
		t.Error("picking a slot kept bob unavailable")
	}

	counts, err := repo.AvailabilityCounts(ctx, t1, week)
	must(t, err)
	if !slices.Contains(counts, availability.SlotCount{DayIndex: 0, SlotIndex: 1, Count: 2}) {
		t.Errorf("counts: %v, want 2 members on Monday slot 1", counts)
	}
	users, err := repo.AvailabilityUsers(ctx, t1, week)
	must(t, err)
	for _, u := range users {
		if u.DayIndex == 0 && u.SlotIndex == 1 && !slices.Equal(u.UserIDs, []string{"alice", "bob"}) {
			t.Errorf("Monday slot 1: %v", u.UserIDs)
		}
	}

	must(t, repo.DeleteWeekAvailability(ctx, t1, week))
	if got, _ := repo.UserAvailability(ctx, t2, week, "alice"); len(got) != 1 {
		t.Errorf("resetting team 1 touched team 2: %v", got)
	}

	// Weekly message: none, then upserted.
	if msg, err := repo.WeeklyMessage(ctx, t1); err != nil || msg != nil {
		t.Errorf("no weekly message yet: %v, %v", msg, err)
	}
	must(t, repo.SetWeeklyMessage(ctx, t1, availability.WeeklyMessage{Week: week, ChannelID: "c1", MessageID: "m1"}))
	must(t, repo.SetWeeklyMessage(ctx, t1, availability.WeeklyMessage{Week: "2026-W43", ChannelID: "c1", MessageID: "m2"}))
	if msg, err := repo.WeeklyMessage(ctx, t1); err != nil || msg.MessageID != "m2" || msg.Week != "2026-W43" {
		t.Errorf("weekly message: %+v, %v", msg, err)
	}

	// Timezones are per member, across teams.
	if zone, err := repo.UserTimezone(ctx, "alice"); err != nil || zone != "" {
		t.Errorf("no timezone yet: %q, %v", zone, err)
	}
	must(t, repo.SetUserTimezone(ctx, "alice", "UTC+9"))
	must(t, repo.SetUserTimezone(ctx, "alice", "UTC-5"))
	if zone, _ := repo.UserTimezone(ctx, "alice"); zone != "UTC-5" {
		t.Errorf("timezone: %q", zone)
	}
}

// Two teams of a guild share nothing: not their availabilities, nor their weekly message.
func TestTeamsAreSeparate(t *testing.T) {
	ctx := t.Context()
	pool := testPool(t)
	teams, avail := NewTeamRepository(pool), NewAvailabilityRepository(pool)
	a, err := teams.Create(ctx, "g1", "A", i18n.FR)
	must(t, err)
	b, err := teams.Create(ctx, "g1", "B", i18n.EN)
	must(t, err)
	const week = availability.WeekKey("2026-W42")

	must(t, avail.SetDayAvailability(ctx, a.ID, week, "alice", 0, []int{1}))
	must(t, avail.SetWeekUnavailable(ctx, b.ID, week, "alice"))
	must(t, avail.SetWeeklyMessage(ctx, a.ID, availability.WeeklyMessage{Week: week, ChannelID: "ca", MessageID: "ma"}))
	must(t, avail.SetWeeklyMessage(ctx, b.ID, availability.WeeklyMessage{Week: week, ChannelID: "cb", MessageID: "mb"}))

	if got, _ := avail.UserAvailability(ctx, b.ID, week, "alice"); len(got) != 0 {
		t.Errorf("alice's slots of team A show in team B: %v", got)
	}
	if unavailable, _ := avail.IsUserWeekUnavailable(ctx, a.ID, week, "alice"); unavailable {
		t.Error("alice unavailable in team B is unavailable in team A")
	}
	if msg, _ := avail.WeeklyMessage(ctx, a.ID); msg.MessageID != "ma" {
		t.Errorf("team A's message: %+v", msg)
	}

	// Deleting a team takes what belongs to it, and only that.
	must(t, teams.Delete(ctx, a.ID))
	if got, _ := avail.AvailabilityCounts(ctx, a.ID, week); len(got) != 0 {
		t.Errorf("slots of the deleted team: %v", got)
	}
	if msg, _ := avail.WeeklyMessage(ctx, a.ID); msg != nil {
		t.Errorf("message of the deleted team: %+v", msg)
	}
	if msg, _ := avail.WeeklyMessage(ctx, b.ID); msg == nil || msg.MessageID != "mb" {
		t.Errorf("team B's message: %+v", msg)
	}
}

func TestTeamRepository(t *testing.T) {
	ctx := t.Context()
	repo := NewTeamRepository(testPool(t))

	created, err := repo.Create(ctx, "g1", "Team A", i18n.EN)
	must(t, err)
	want := team.Team{ID: created.ID, GuildID: "g1", Name: "Team A", SlotLayout: availability.HalfDayLayout(), Display: team.DisplayClassic, Language: i18n.EN, Schedule: availability.DefaultSchedule(), Active: true, Location: availability.DefaultLocation()}
	if *created != want {
		t.Errorf("defaults: got %+v, want %+v", *created, want)
	}
	if _, err := repo.Get(ctx, created.ID+1000); !errors.Is(err, team.ErrNotFound) {
		t.Errorf("unknown team: %v", err)
	}

	// A name is unique within a guild only.
	if _, err := repo.Create(ctx, "g1", "Team A", i18n.FR); !errors.Is(err, team.ErrNameTaken) {
		t.Errorf("same name in the guild: %v", err)
	}
	other, err := repo.Create(ctx, "g2", "Team A", i18n.FR)
	must(t, err)
	second, err := repo.Create(ctx, "g1", "Team B", i18n.FR)
	must(t, err)
	if err := repo.Rename(ctx, second.ID, "Team A"); !errors.Is(err, team.ErrNameTaken) {
		t.Errorf("renaming to a taken name: %v", err)
	}
	must(t, repo.Rename(ctx, second.ID, "Team C"))
	if got, _ := repo.Get(ctx, second.ID); got.Name != "Team C" {
		t.Errorf("renamed: %q", got.Name)
	}
	list, err := repo.List(ctx, "g1")
	must(t, err)
	if len(list) != 2 || list[0].ID != created.ID || list[1].ID != second.ID {
		t.Errorf("g1's teams, oldest first: %+v", list)
	}

	// Each team has its own settings.
	layout, _ := availability.HourlyLayout(20, 24)
	schedule, _ := availability.NewSchedule(time.Sunday, 18)
	must(t, repo.SetChannel(ctx, created.ID, "c1"))
	must(t, repo.SetRole(ctx, created.ID, "r1"))
	must(t, repo.SetMessageSettings(ctx, created.ID, layout, team.DisplayImage, i18n.FR))
	must(t, repo.SetSchedule(ctx, created.ID, schedule))

	got, err := repo.Get(ctx, created.ID)
	must(t, err)
	want = team.Team{ID: created.ID, GuildID: "g1", Name: "Team A", ChannelID: "c1", RoleID: "r1", SlotLayout: layout, Display: team.DisplayImage, Language: i18n.FR, Schedule: schedule, Active: true, Location: availability.DefaultLocation()}
	if *got != want {
		t.Errorf("saved: got %+v, want %+v", *got, want)
	}
	if got, _ := repo.Get(ctx, second.ID); got.Display != team.DisplayClassic || got.SlotLayout.IsHourly() || got.ChannelID != "" {
		t.Errorf("the settings leaked to the guild's other team: %+v", got)
	}

	withChannel, err := repo.ListWithChannel(ctx)
	must(t, err)
	if len(withChannel) != 1 || *withChannel[0] != want {
		t.Errorf("teams with a channel: %v", withChannel)
	}

	// A paused team keeps its settings and is not published until it comes back.
	must(t, repo.SetActive(ctx, created.ID, false))
	if list, _ := repo.ListWithChannel(ctx); len(list) != 0 {
		t.Errorf("paused team still listed: %v", list)
	}
	got, err = repo.Get(ctx, created.ID)
	must(t, err)
	if want.Active = false; *got != want {
		t.Errorf("paused: got %+v, want %+v", *got, want)
	}

	// Timezones: a country's (follows daylight saving time) or a fixed offset.
	for _, zone := range []string{"America/Toronto", "UTC+5:30"} {
		loc, err := availability.ParseTimezone(zone)
		must(t, err)
		must(t, repo.SetTimezone(ctx, second.ID, loc))
		if got, err := repo.Get(ctx, second.ID); err != nil || got.Location.String() != zone {
			t.Errorf("timezone %s: got %v, %v", zone, got.Location, err)
		}
	}
	must(t, repo.SetFullAt(ctx, second.ID, 4))
	if got, _ := repo.Get(ctx, second.ID); got.FullAt != 4 {
		t.Errorf("full at: %d, want 4", got.FullAt)
	}
	must(t, repo.SetFullAt(ctx, second.ID, 0)) // back to relative
	if got, _ := repo.Get(ctx, second.ID); got.FullAt != 0 {
		t.Errorf("full at after reset: %d, want 0", got.FullAt)
	}
	must(t, repo.SetMemberTimezones(ctx, second.ID, true))
	if got, _ := repo.Get(ctx, second.ID); !got.MemberTimezones {
		t.Error("member timezones not saved")
	}
	must(t, repo.SetChannel(ctx, created.ID, "c2")) // setting a channel resumes it
	if list, _ := repo.ListWithChannel(ctx); len(list) != 1 || list[0].ChannelID != "c2" || !list[0].Active {
		t.Errorf("resumed team: %v", list)
	}

	// The bot leaving a guild pauses all its teams, and only those.
	ids, err := repo.SetGuildActive(ctx, "g1", false)
	must(t, err)
	if len(ids) != 2 {
		t.Errorf("paused teams: %v", ids)
	}
	if ids, _ := repo.SetGuildActive(ctx, "g1", false); len(ids) != 0 {
		t.Errorf("nothing left to pause: %v", ids)
	}
	if got, _ := repo.Get(ctx, other.ID); !got.Active {
		t.Error("pausing g1 touched g2")
	}
}

func TestGuildRepository(t *testing.T) {
	ctx := t.Context()
	repo := NewGuildRepository(testPool(t))

	if g, err := repo.Get(ctx, "g1"); err != nil || g.Welcome != nil || g.Language != "" {
		t.Errorf("nothing saved yet: %+v, %v", g, err)
	}
	must(t, repo.SetWelcome(ctx, "g1", guild.Welcome{ChannelID: "c1", MessageID: "m1"}))
	must(t, repo.SetWelcome(ctx, "g1", guild.Welcome{ChannelID: "c2", MessageID: "m2"}))
	if g, err := repo.Get(ctx, "g1"); err != nil || *g.Welcome != (guild.Welcome{ChannelID: "c2", MessageID: "m2"}) || g.Language != "" {
		t.Errorf("welcome: %+v, %v", g, err)
	}

	// The language and the welcome message are saved apart, and the language outlives it.
	must(t, repo.SetLanguage(ctx, "g1", i18n.EN))
	must(t, repo.SetLanguage(ctx, "g2", i18n.FR)) // a guild without welcome message
	must(t, repo.ForgetWelcome(ctx, "g1"))
	if g, _ := repo.Get(ctx, "g1"); g.Welcome != nil || g.Language != i18n.EN {
		t.Errorf("forgotten welcome: %+v", g)
	}
	if g, _ := repo.Get(ctx, "g2"); g.Welcome != nil || g.Language != i18n.FR {
		t.Errorf("language only: %+v", g)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
