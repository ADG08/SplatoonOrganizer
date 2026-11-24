package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation is Postgres' error code for a duplicate key.
const uniqueViolation = "23505"

// TeamRepository implements team.Repository using Postgres.
type TeamRepository struct {
	queries *db.Queries
}

// NewTeamRepository creates a new Postgres team repository.
func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{
		queries: db.New(pool),
	}
}

// Ensure TeamRepository implements the interface.
var _ appteam.Repository = (*TeamRepository)(nil)

// nameError translates the duplicate name of a guild into team.ErrNameTaken.
func nameError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return team.ErrNameTaken
	}
	return err
}

func (r *TeamRepository) Create(ctx context.Context, guildID, name string, lang i18n.Lang) (*team.Team, error) {
	row, err := r.queries.CreateTeam(ctx, db.CreateTeamParams{
		GuildID:  guildID,
		Name:     name,
		Language: pgtype.Text{String: string(lang), Valid: lang != ""},
	})
	if err != nil {
		return nil, nameError(err)
	}
	return toTeam(row)
}

func (r *TeamRepository) Get(ctx context.Context, id int64) (*team.Team, error) {
	row, err := r.queries.GetTeam(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, team.ErrNotFound
		}
		return nil, err
	}
	return toTeam(row)
}

func (r *TeamRepository) List(ctx context.Context, guildID string) ([]*team.Team, error) {
	rows, err := r.queries.ListTeams(ctx, guildID)
	if err != nil {
		return nil, err
	}
	return toTeams(rows)
}

func (r *TeamRepository) ListWithChannel(ctx context.Context) ([]*team.Team, error) {
	rows, err := r.queries.ListTeamsWithChannel(ctx)
	if err != nil {
		return nil, err
	}
	return toTeams(rows)
}

func toTeams(rows []db.Team) ([]*team.Team, error) {
	teams := make([]*team.Team, 0, len(rows))
	for _, row := range rows {
		t, err := toTeam(row)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func toTeam(row db.Team) (*team.Team, error) {
	t := &team.Team{
		ID:              row.ID,
		GuildID:         row.GuildID,
		Name:            row.Name,
		SlotLayout:      availability.HalfDayLayout(),
		Display:         team.DisplayClassic,
		Language:        i18n.Parse(row.Language.String),
		Schedule:        availability.DefaultSchedule(),
		Active:          row.Active,
		MemberTimezones: row.MemberTimezones,
		FullAt:          int(row.FullAt.Int16),
		Location:        availability.DefaultLocation(),
	}
	if row.Timezone.Valid {
		loc, err := availability.ParseTimezone(row.Timezone.String)
		if err != nil {
			return nil, fmt.Errorf("team %d: stored timezone: %w", row.ID, err)
		}
		t.Location = loc
	}
	if team.Display(row.DisplayStyle.String) == team.DisplayImage {
		t.Display = team.DisplayImage
	}
	if row.ChannelID.Valid {
		t.ChannelID = row.ChannelID.String
	}
	if row.RoleID.Valid {
		t.RoleID = row.RoleID.String
	}
	if availability.SlotMode(row.SlotMode.String) == availability.ModeHourly {
		layout, err := availability.HourlyLayout(int(row.SlotStartHour.Int16), int(row.SlotEndHour.Int16))
		if err != nil {
			return nil, fmt.Errorf("team %d: stored slot layout: %w", row.ID, err)
		}
		t.SlotLayout = layout
	}
	if row.PublishWeekday.Valid && row.PublishHour.Valid {
		schedule, err := availability.NewSchedule(time.Weekday(row.PublishWeekday.Int16), int(row.PublishHour.Int16))
		if err != nil {
			return nil, fmt.Errorf("team %d: stored publication schedule: %w", row.ID, err)
		}
		t.Schedule = schedule
	}
	return t, nil
}

func (r *TeamRepository) Delete(ctx context.Context, id int64) error {
	return r.queries.DeleteTeam(ctx, id)
}

func (r *TeamRepository) Rename(ctx context.Context, id int64, name string) error {
	return nameError(r.queries.RenameTeam(ctx, db.RenameTeamParams{ID: id, Name: name}))
}

func (r *TeamRepository) SetMessageSettings(ctx context.Context, id int64, layout availability.Layout, display team.Display, lang i18n.Lang) error {
	params := db.SetTeamMessageSettingsParams{
		ID:           id,
		SlotMode:     pgtype.Text{String: string(layout.Mode), Valid: layout.Mode != ""},
		DisplayStyle: pgtype.Text{String: string(display), Valid: display != ""},
		Language:     pgtype.Text{String: string(lang), Valid: lang != ""},
	}
	if layout.IsHourly() {
		params.SlotStartHour = pgtype.Int2{Int16: int16(layout.StartHour), Valid: true}
		params.SlotEndHour = pgtype.Int2{Int16: int16(layout.EndHour), Valid: true}
	}
	return r.queries.SetTeamMessageSettings(ctx, params)
}

func (r *TeamRepository) SetSchedule(ctx context.Context, id int64, schedule availability.Schedule) error {
	return r.queries.SetTeamSchedule(ctx, db.SetTeamScheduleParams{
		ID:             id,
		PublishWeekday: pgtype.Int2{Int16: int16(schedule.Weekday), Valid: true},
		PublishHour:    pgtype.Int2{Int16: int16(schedule.Hour), Valid: true},
	})
}

func (r *TeamRepository) SetTimezone(ctx context.Context, id int64, loc *time.Location) error {
	return r.queries.SetTeamTimezone(ctx, db.SetTeamTimezoneParams{
		ID:       id,
		Timezone: pgtype.Text{String: loc.String(), Valid: true},
	})
}

func (r *TeamRepository) SetMemberTimezones(ctx context.Context, id int64, on bool) error {
	return r.queries.SetTeamMemberTimezones(ctx, db.SetTeamMemberTimezonesParams{ID: id, MemberTimezones: on})
}

func (r *TeamRepository) SetFullAt(ctx context.Context, id int64, fullAt int) error {
	return r.queries.SetTeamFullAt(ctx, db.SetTeamFullAtParams{
		ID:     id,
		FullAt: pgtype.Int2{Int16: int16(fullAt), Valid: fullAt > 0},
	})
}

func (r *TeamRepository) SetActive(ctx context.Context, id int64, active bool) error {
	return r.queries.SetTeamActive(ctx, db.SetTeamActiveParams{ID: id, Active: active})
}

func (r *TeamRepository) SetGuildActive(ctx context.Context, guildID string, active bool) ([]int64, error) {
	return r.queries.SetGuildTeamsActive(ctx, db.SetGuildTeamsActiveParams{GuildID: guildID, Active: active})
}

func (r *TeamRepository) SetChannel(ctx context.Context, id int64, channelID string) error {
	return r.queries.SetTeamChannel(ctx, db.SetTeamChannelParams{
		ID:        id,
		ChannelID: pgtype.Text{String: channelID, Valid: channelID != ""},
	})
}

func (r *TeamRepository) SetRole(ctx context.Context, id int64, roleID string) error {
	return r.queries.SetTeamRole(ctx, db.SetTeamRoleParams{
		ID:     id,
		RoleID: pgtype.Text{String: roleID, Valid: roleID != ""},
	})
}
