package team

import (
	"context"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Repository is the outbound port for team persistence. Create and Rename return
// team.ErrNameTaken for a name already used in the guild; Get returns team.ErrNotFound.
type Repository interface {
	Create(ctx context.Context, guildID, name string, lang i18n.Lang) (*team.Team, error)
	Get(ctx context.Context, id int64) (*team.Team, error)
	// List returns the guild's teams, oldest first.
	List(ctx context.Context, guildID string) ([]*team.Team, error)
	// ListWithChannel returns the active teams that configured a channel for the weekly message.
	ListWithChannel(ctx context.Context) ([]*team.Team, error)
	Delete(ctx context.Context, id int64) error
	Rename(ctx context.Context, id int64, name string) error
	// SetChannel also makes the team active again.
	SetChannel(ctx context.Context, id int64, channelID string) error
	SetActive(ctx context.Context, id int64, active bool) error
	// SetGuildActive changes every team of the guild, and returns the ones that changed.
	SetGuildActive(ctx context.Context, guildID string, active bool) ([]int64, error)
	SetRole(ctx context.Context, id int64, roleID string) error
	SetMessageSettings(ctx context.Context, id int64, layout availability.Layout, display team.Display, lang i18n.Lang) error
	SetSchedule(ctx context.Context, id int64, schedule availability.Schedule) error
	SetTimezone(ctx context.Context, id int64, loc *time.Location) error
	SetMemberTimezones(ctx context.Context, id int64, on bool) error
	// SetFullAt sets the players from which a slot is full; 0 makes it relative again.
	SetFullAt(ctx context.Context, id int64, fullAt int) error
}
