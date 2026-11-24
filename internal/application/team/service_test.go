package team

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// fakeRepo stores teams by ID. When paused is set, the next Get reads the team, signals
// it on paused, then waits for resume before returning it.
type fakeRepo struct {
	mu             sync.Mutex
	teams          map[int64]team.Team
	paused, resume chan struct{}
}

func newFakeRepo(teams ...team.Team) *fakeRepo {
	r := &fakeRepo{teams: make(map[int64]team.Team)}
	for _, t := range teams {
		r.teams[t.ID] = t
	}
	return r
}

func (r *fakeRepo) Get(_ context.Context, id int64) (*team.Team, error) {
	r.mu.Lock()
	t, ok := r.teams[id]
	paused, resume := r.paused, r.resume
	r.paused = nil
	r.mu.Unlock()
	if !ok {
		return nil, team.ErrNotFound
	}
	if paused != nil {
		close(paused)
		<-resume
	}
	return &t, nil
}

func (r *fakeRepo) set(id int64, change func(*team.Team)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.teams[id]
	change(&t)
	r.teams[id] = t
	return nil
}

func (r *fakeRepo) List(_ context.Context, guildID string) ([]*team.Team, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var teams []*team.Team
	for id := int64(1); id <= int64(len(r.teams)); id++ {
		if t := r.teams[id]; t.GuildID == guildID {
			teams = append(teams, &t)
		}
	}
	return teams, nil
}

func (r *fakeRepo) Create(_ context.Context, guildID, name string, lang i18n.Lang) (*team.Team, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := team.Team{ID: int64(len(r.teams) + 1), GuildID: guildID, Name: name, Language: lang}
	r.teams[t.ID] = t
	return &t, nil
}

func (r *fakeRepo) ListWithChannel(context.Context) ([]*team.Team, error) { return nil, nil }
func (r *fakeRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.teams, id)
	return nil
}
func (r *fakeRepo) Rename(_ context.Context, id int64, name string) error {
	return r.set(id, func(t *team.Team) { t.Name = name })
}
func (r *fakeRepo) SetChannel(_ context.Context, id int64, channelID string) error {
	return r.set(id, func(t *team.Team) { t.ChannelID = channelID })
}
func (r *fakeRepo) SetRole(_ context.Context, id int64, roleID string) error {
	return r.set(id, func(t *team.Team) { t.RoleID = roleID })
}
func (r *fakeRepo) SetMemberTimezones(_ context.Context, id int64, on bool) error {
	return r.set(id, func(t *team.Team) { t.MemberTimezones = on })
}
func (r *fakeRepo) SetFullAt(_ context.Context, id int64, fullAt int) error {
	return r.set(id, func(t *team.Team) { t.FullAt = fullAt })
}
func (r *fakeRepo) SetActive(_ context.Context, id int64, active bool) error {
	return r.set(id, func(t *team.Team) { t.Active = active })
}
func (r *fakeRepo) SetGuildActive(_ context.Context, guildID string, active bool) ([]int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var changed []int64
	for id, t := range r.teams {
		if t.GuildID == guildID && t.Active != active {
			t.Active = active
			r.teams[id] = t
			changed = append(changed, id)
		}
	}
	return changed, nil
}
func (r *fakeRepo) SetTimezone(_ context.Context, id int64, loc *time.Location) error {
	return r.set(id, func(t *team.Team) { t.Location = loc })
}
func (r *fakeRepo) SetSchedule(_ context.Context, id int64, s availability.Schedule) error {
	return r.set(id, func(t *team.Team) { t.Schedule = s })
}
func (r *fakeRepo) SetMessageSettings(_ context.Context, id int64, l availability.Layout, d team.Display, lang i18n.Lang) error {
	return r.set(id, func(t *team.Team) { t.SlotLayout, t.Display, t.Language = l, d, lang })
}

// A read that started before a change must not leave the old team in the cache.
func TestCacheKeepsChangeMadeDuringRead(t *testing.T) {
	repo := newFakeRepo(team.Team{ID: 1, GuildID: "g1", RoleID: "old"})
	repo.paused, repo.resume = make(chan struct{}), make(chan struct{})
	svc := NewService(repo)
	ctx := t.Context()

	paused := repo.paused
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := svc.Get(ctx, 1); err != nil {
			t.Error(err)
		}
	}()
	<-paused // the read got the old team and is not cached yet

	if err := svc.SetRole(ctx, 1, "new"); err != nil {
		t.Fatal(err)
	}
	close(repo.resume)
	<-done

	if got, _ := svc.Get(ctx, 1); got.RoleID != "new" {
		t.Errorf("cached role %q after the change, want %q", got.RoleID, "new")
	}
}

// Team IDs come from custom IDs: a team of another guild must look like it doesn't exist.
func TestGetInGuild(t *testing.T) {
	svc := NewService(newFakeRepo(team.Team{ID: 1, GuildID: "g1"}))
	ctx := t.Context()
	if _, err := svc.GetInGuild(ctx, "g1", 1); err != nil {
		t.Errorf("own team: %v", err)
	}
	if _, err := svc.GetInGuild(ctx, "g2", 1); !errors.Is(err, team.ErrNotFound) {
		t.Errorf("other guild's team: %v", err)
	}
}

func TestPrimaryIsOldestTeam(t *testing.T) {
	svc := NewService(newFakeRepo(team.Team{ID: 1, GuildID: "g2"}, team.Team{ID: 2, GuildID: "g1"}, team.Team{ID: 3, GuildID: "g1"}))
	ctx := t.Context()
	if got, err := svc.Primary(ctx, "g1"); err != nil || got.ID != 2 {
		t.Errorf("primary: %+v, %v", got, err)
	}
	if _, err := svc.Primary(ctx, "g3"); !errors.Is(err, team.ErrNotFound) {
		t.Errorf("guild without team: %v", err)
	}
}

func TestCreateLimit(t *testing.T) {
	svc := NewService(newFakeRepo())
	ctx := t.Context()
	for n := range team.MaxTeamsPerGuild {
		created, err := svc.Create(ctx, "g1", fmt.Sprintf(" Team %d ", n), i18n.EN)
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf("Team %d", n); created.Name != want || created.Language != i18n.EN {
			t.Errorf("created %+v, want name %q in English", created, want)
		}
	}
	if _, err := svc.Create(ctx, "g1", "One too many", i18n.FR); !errors.Is(err, team.ErrTooManyTeams) {
		t.Errorf("over the limit: %v", err)
	}
	if _, err := svc.Create(ctx, "g2", "  ", i18n.FR); !errors.Is(err, team.ErrNameEmpty) {
		t.Errorf("empty name: %v", err)
	}
}

// Pausing a guild must reach the cached teams, as the scheduler and the panel read them.
func TestSetGuildActive(t *testing.T) {
	svc := NewService(newFakeRepo(team.Team{ID: 1, GuildID: "g1", Active: true}, team.Team{ID: 2, GuildID: "g1"}, team.Team{ID: 3, GuildID: "g2", Active: true}))
	ctx := t.Context()
	if _, err := svc.Get(ctx, 1); err != nil { // cached
		t.Fatal(err)
	}
	if changed, err := svc.SetGuildActive(ctx, "g1", false); err != nil || !changed {
		t.Fatalf("pausing: changed %v, %v", changed, err)
	}
	if got, _ := svc.Get(ctx, 1); got.Active {
		t.Error("the cached team stayed active")
	}
	if got, _ := svc.Get(ctx, 3); !got.Active {
		t.Error("pausing g1 touched g2")
	}
	if changed, _ := svc.SetGuildActive(ctx, "g1", false); changed {
		t.Error("nothing to pause, yet reported as changed")
	}
}

func TestDeleteForgetsTeam(t *testing.T) {
	svc := NewService(newFakeRepo(team.Team{ID: 1, GuildID: "g1"}))
	ctx := t.Context()
	if _, err := svc.Get(ctx, 1); err != nil { // cached
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 1); !errors.Is(err, team.ErrNotFound) {
		t.Errorf("deleted team: %v", err)
	}
}
