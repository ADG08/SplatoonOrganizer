package postgres

import (
	"context"
	"errors"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
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
			return &guildconfig.GuildConfig{}, nil
		}
		return nil, err
	}

	cfg := &guildconfig.GuildConfig{}
	if row.ChannelID.Valid {
		cfg.ChannelID = row.ChannelID.String
	}
	if row.RoleID.Valid {
		cfg.RoleID = row.RoleID.String
	}
	return cfg, nil
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
