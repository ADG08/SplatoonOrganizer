package guildconfig

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// fakeRepo stores one config. When paused is set, the next Get reads the config, signals
// it on paused, then waits for resume before returning it.
type fakeRepo struct {
	mu             sync.Mutex
	cfg            guildconfig.GuildConfig
	paused, resume chan struct{}
}

func (r *fakeRepo) Get(context.Context, string) (*guildconfig.GuildConfig, error) {
	r.mu.Lock()
	cfg, paused, resume := r.cfg, r.paused, r.resume
	r.paused = nil
	r.mu.Unlock()
	if paused != nil {
		close(paused)
		<-resume
	}
	return &cfg, nil
}

func (r *fakeRepo) set(change func(*guildconfig.GuildConfig)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	change(&r.cfg)
	return nil
}

func (r *fakeRepo) ListWithChannel(context.Context) ([]*guildconfig.GuildConfig, error) {
	return nil, nil
}
func (r *fakeRepo) SetChannel(_ context.Context, _, id string) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.ChannelID = id })
}
func (r *fakeRepo) SetRole(_ context.Context, _, id string) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.RoleID = id })
}
func (r *fakeRepo) SetMemberTimezones(_ context.Context, _ string, on bool) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.MemberTimezones = on })
}
func (r *fakeRepo) SetFullAt(_ context.Context, _ string, fullAt int) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.FullAt = fullAt })
}
func (r *fakeRepo) SetActive(_ context.Context, _ string, active bool) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.Active = active })
}
func (r *fakeRepo) SetTimezone(_ context.Context, _ string, loc *time.Location) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.Location = loc })
}
func (r *fakeRepo) SetSchedule(_ context.Context, _ string, s availability.Schedule) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.Schedule = s })
}
func (r *fakeRepo) SetMessageSettings(_ context.Context, _ string, l availability.Layout, d guildconfig.Display, lang i18n.Lang) error {
	return r.set(func(c *guildconfig.GuildConfig) { c.SlotLayout, c.Display, c.Language = l, d, lang })
}

// A read that started before a change must not leave the old config in the cache.
func TestCacheKeepsChangeMadeDuringRead(t *testing.T) {
	repo := &fakeRepo{cfg: guildconfig.GuildConfig{GuildID: "g1", RoleID: "old"}}
	repo.paused, repo.resume = make(chan struct{}), make(chan struct{})
	svc := NewService(repo)
	ctx := t.Context()

	paused := repo.paused
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := svc.Get(ctx, "g1"); err != nil {
			t.Error(err)
		}
	}()
	<-paused // the read got the old config and is not cached yet

	if err := svc.SetRole(ctx, "g1", "new"); err != nil {
		t.Fatal(err)
	}
	close(repo.resume)
	<-done

	if cfg, _ := svc.Get(ctx, "g1"); cfg.RoleID != "new" {
		t.Errorf("cached role %q after the change, want %q", cfg.RoleID, "new")
	}
}
