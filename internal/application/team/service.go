package team

import (
	"context"
	"sync"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Service implements team use cases.
type Service struct {
	repo Repository

	// cache holds the teams already read: they are needed on nearly every interaction
	// and only change through this service (single bot process).
	cache sync.Map // team ID -> team.Team
	// writes serializes the changes, each one caching the team it leaves in the database.
	writes sync.Mutex
}

// NewService creates a new team service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns a team. The returned team is a copy the caller may modify.
func (s *Service) Get(ctx context.Context, id int64) (*team.Team, error) {
	if t, ok := s.cache.Load(id); ok {
		c := t.(team.Team)
		return &c, nil
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// A change saved while this one was read cached the newer team: keep that one.
	cached, _ := s.cache.LoadOrStore(id, *t)
	c := cached.(team.Team)
	return &c, nil
}

// GetInGuild returns a team of the guild, else team.ErrNotFound: team IDs come from
// custom IDs, so they are checked against the interaction's guild.
func (s *Service) GetInGuild(ctx context.Context, guildID string, id int64) (*team.Team, error) {
	t, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.GuildID != guildID {
		return nil, team.ErrNotFound
	}
	return t, nil
}

// Primary returns the guild's oldest team, else team.ErrNotFound. It owns the messages
// posted before the guild had teams: it took over the guild's configuration.
func (s *Service) Primary(ctx context.Context, guildID string) (*team.Team, error) {
	teams, err := s.repo.List(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, team.ErrNotFound
	}
	return teams[0], nil
}

// List returns the guild's teams, oldest first.
func (s *Service) List(ctx context.Context, guildID string) ([]*team.Team, error) {
	return s.repo.List(ctx, guildID)
}

// ListWithChannel returns the active teams that configured a channel for the weekly message.
func (s *Service) ListWithChannel(ctx context.Context) ([]*team.Team, error) {
	return s.repo.ListWithChannel(ctx)
}

// Create adds a team to the guild, within the limit of teams per guild.
func (s *Service) Create(ctx context.Context, guildID, name string, lang i18n.Lang) (*team.Team, error) {
	name, err := team.NormalizeName(name)
	if err != nil {
		return nil, err
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	teams, err := s.repo.List(ctx, guildID)
	if err != nil {
		return nil, err
	}
	if len(teams) >= team.MaxTeamsPerGuild {
		return nil, team.ErrTooManyTeams
	}
	return s.repo.Create(ctx, guildID, name, lang)
}

// Delete removes a team with its availabilities.
func (s *Service) Delete(ctx context.Context, id int64) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	s.cache.Delete(id)
	return s.repo.Delete(ctx, id)
}

// update saves a change, then caches the team as it now is in the database. Caching it
// (rather than forgetting it) keeps a read that started before the change from caching
// the old team.
func (s *Service) update(ctx context.Context, id int64, write func() error) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := write(); err != nil {
		s.cache.Delete(id)
		return err
	}
	return s.recache(ctx, id)
}

// recache caches the team as it is in the database.
func (s *Service) recache(ctx context.Context, id int64) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		s.cache.Delete(id)
		return err
	}
	s.cache.Store(id, *t)
	return nil
}

// Rename changes a team's name.
func (s *Service) Rename(ctx context.Context, id int64, name string) error {
	name, err := team.NormalizeName(name)
	if err != nil {
		return err
	}
	return s.update(ctx, id, func() error { return s.repo.Rename(ctx, id, name) })
}

// SetChannel updates the team's message channel, and makes it active again.
func (s *Service) SetChannel(ctx context.Context, id int64, channelID string) error {
	return s.update(ctx, id, func() error { return s.repo.SetChannel(ctx, id, channelID) })
}

// SetTimezone updates the timezone of the team's hours.
func (s *Service) SetTimezone(ctx context.Context, id int64, loc *time.Location) error {
	return s.update(ctx, id, func() error { return s.repo.SetTimezone(ctx, id, loc) })
}

// SetMemberTimezones turns on or off members seeing the slots in their own timezone.
func (s *Service) SetMemberTimezones(ctx context.Context, id int64, on bool) error {
	return s.update(ctx, id, func() error { return s.repo.SetMemberTimezones(ctx, id, on) })
}

// SetFullAt sets the players from which a slot is shown full; 0 rates the slots against
// the week's best one.
func (s *Service) SetFullAt(ctx context.Context, id int64, fullAt int) error {
	return s.update(ctx, id, func() error { return s.repo.SetFullAt(ctx, id, fullAt) })
}

// SetActive turns the publication of the weekly message on or off for a team, keeping
// its settings.
func (s *Service) SetActive(ctx context.Context, id int64, active bool) error {
	return s.update(ctx, id, func() error { return s.repo.SetActive(ctx, id, active) })
}

// SetGuildActive turns the publication on or off for every team of the guild, keeping
// their settings, and reports whether one changed.
func (s *Service) SetGuildActive(ctx context.Context, guildID string, active bool) (bool, error) {
	s.writes.Lock()
	defer s.writes.Unlock()
	ids, err := s.repo.SetGuildActive(ctx, guildID, active)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if err := s.recache(ctx, id); err != nil {
			return true, err
		}
	}
	return len(ids) > 0, nil
}

// SetRole updates the role to ping for a team.
func (s *Service) SetRole(ctx context.Context, id int64, roleID string) error {
	return s.update(ctx, id, func() error { return s.repo.SetRole(ctx, id, roleID) })
}

// SetMessageSettings updates how days are split into slots, how the weekly message is
// shown and the language of the bot for the team.
func (s *Service) SetMessageSettings(ctx context.Context, id int64, layout availability.Layout, display team.Display, lang i18n.Lang) error {
	return s.update(ctx, id, func() error { return s.repo.SetMessageSettings(ctx, id, layout, display, lang) })
}

// SetSchedule updates when the team's weekly message is published.
func (s *Service) SetSchedule(ctx context.Context, id int64, schedule availability.Schedule) error {
	return s.update(ctx, id, func() error { return s.repo.SetSchedule(ctx, id, schedule) })
}
