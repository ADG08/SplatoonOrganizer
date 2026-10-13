package postgres

import (
	"context"
	"errors"
	"fmt"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
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
			return &guildconfig.GuildConfig{SlotLayout: availability.HalfDayLayout(), Display: guildconfig.DisplayClassic}, nil
		}
		return nil, err
	}

	cfg := &guildconfig.GuildConfig{SlotLayout: availability.HalfDayLayout(), Display: guildconfig.DisplayClassic}
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
			return nil, fmt.Errorf("guild %s: stored slot layout: %w", guildID, err)
		}
		cfg.SlotLayout = layout
	}
	return cfg, nil
}

func (r *GuildConfigRepository) SetSlotSettings(ctx context.Context, guildID string, layout availability.Layout, display guildconfig.Display) error {
	params := db.SetGuildConfigSlotSettingsParams{
		GuildID:      guildID,
		SlotMode:     pgtype.Text{String: string(layout.Mode), Valid: layout.Mode != ""},
		DisplayStyle: pgtype.Text{String: string(display), Valid: display != ""},
	}
	if layout.IsHourly() {
		params.SlotStartHour = pgtype.Int2{Int16: int16(layout.StartHour), Valid: true}
		params.SlotEndHour = pgtype.Int2{Int16: int16(layout.EndHour), Valid: true}
	}
	return r.queries.SetGuildConfigSlotSettings(ctx, params)
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
