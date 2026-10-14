package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GuildConfigRepository implements guildconfig.Repository using Postgres.
type GuildConfigRepository struct {
	queries *db.Queries
}

// NewGuildConfigRepository creates a new Postgres guild config repository.
func NewGuildConfigRepository(pool *pgxpool.Pool) *GuildConfigRepository {
	return &GuildConfigRepository{
		queries: db.New(pool),
	}
}

// Ensure GuildConfigRepository implements the interface.
var _ appguildconfig.Repository = (*GuildConfigRepository)(nil)

func (r *GuildConfigRepository) Get(ctx context.Context, guildID string) (*guildconfig.GuildConfig, error) {
	row, err := r.queries.GetGuildConfig(ctx, guildID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &guildconfig.GuildConfig{GuildID: guildID, SlotLayout: availability.HalfDayLayout(), Display: guildconfig.DisplayClassic, Language: i18n.FR, Schedule: availability.DefaultSchedule(), Active: true}, nil
		}
		return nil, err
	}
	return toGuildConfig(row)
}

func (r *GuildConfigRepository) ListWithChannel(ctx context.Context) ([]*guildconfig.GuildConfig, error) {
	rows, err := r.queries.ListGuildConfigsWithChannel(ctx)
	if err != nil {
		return nil, err
	}
	cfgs := make([]*guildconfig.GuildConfig, 0, len(rows))
	for _, row := range rows {
		cfg, err := toGuildConfig(row)
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

func toGuildConfig(row db.GuildConfig) (*guildconfig.GuildConfig, error) {
	cfg := &guildconfig.GuildConfig{
		GuildID:    row.GuildID,
		SlotLayout: availability.HalfDayLayout(),
		Display:    guildconfig.DisplayClassic,
		Language:   i18n.Parse(row.Language.String),
		Schedule:   availability.DefaultSchedule(),
		Active:     row.Active,
	}
	if guildconfig.Display(row.DisplayStyle.String) == guildconfig.DisplayImage {
		cfg.Display = guildconfig.DisplayImage
	}
	if row.ChannelID.Valid {
		cfg.ChannelID = row.ChannelID.String
	}
	if row.RoleID.Valid {
		cfg.RoleID = row.RoleID.String
	}
	if availability.SlotMode(row.SlotMode.String) == availability.ModeHourly {
		layout, err := availability.HourlyLayout(int(row.SlotStartHour.Int16), int(row.SlotEndHour.Int16))
		if err != nil {
			return nil, fmt.Errorf("guild %s: stored slot layout: %w", row.GuildID, err)
		}
		cfg.SlotLayout = layout
	}
	if row.PublishWeekday.Valid && row.PublishHour.Valid {
		schedule, err := availability.NewSchedule(time.Weekday(row.PublishWeekday.Int16), int(row.PublishHour.Int16))
		if err != nil {
			return nil, fmt.Errorf("guild %s: stored publication schedule: %w", row.GuildID, err)
		}
		cfg.Schedule = schedule
	}
	return cfg, nil
}

func (r *GuildConfigRepository) SetMessageSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display, lang i18n.Lang) error {
	params := db.SetGuildConfigMessageSettingsParams{
		GuildID:      guildID,
		SlotMode:     pgtype.Text{String: string(layout.Mode), Valid: layout.Mode != ""},
		DisplayStyle: pgtype.Text{String: string(display), Valid: display != ""},
		Language:     pgtype.Text{String: string(lang), Valid: lang != ""},
	}
	if layout.IsHourly() {
		params.SlotStartHour = pgtype.Int2{Int16: int16(layout.StartHour), Valid: true}
		params.SlotEndHour = pgtype.Int2{Int16: int16(layout.EndHour), Valid: true}
	}
	return r.queries.SetGuildConfigMessageSettings(ctx, params)
}

func (r *GuildConfigRepository) SetSchedule(ctx context.Context, guildID string, schedule availability.Schedule) error {
	return r.queries.SetGuildConfigSchedule(ctx, db.SetGuildConfigScheduleParams{
		GuildID:        guildID,
		PublishWeekday: pgtype.Int2{Int16: int16(schedule.Weekday), Valid: true},
		PublishHour:    pgtype.Int2{Int16: int16(schedule.Hour), Valid: true},
	})
}

func (r *GuildConfigRepository) SetActive(ctx context.Context, guildID string, active bool) error {
	return r.queries.SetGuildConfigActive(ctx, db.SetGuildConfigActiveParams{GuildID: guildID, Active: active})
}

func (r *GuildConfigRepository) SetChannel(ctx context.Context, guildID, channelID string) error {
	return r.queries.SetGuildConfigChannel(ctx, db.SetGuildConfigChannelParams{
		GuildID:   guildID,
		ChannelID: pgtype.Text{String: channelID, Valid: channelID != ""},
	})
}

func (r *GuildConfigRepository) SetRole(ctx context.Context, guildID, roleID string) error {
	return r.queries.SetGuildConfigRole(ctx, db.SetGuildConfigRoleParams{
		GuildID: guildID,
		RoleID:  pgtype.Text{String: roleID, Valid: roleID != ""},
	})
}
