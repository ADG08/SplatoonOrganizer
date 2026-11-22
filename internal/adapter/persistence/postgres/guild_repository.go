package postgres

import (
	"context"
	"errors"
	"fmt"

	appguild "github.com/ADG08/SplatoonOrganizer/internal/application/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GuildRepository implements guild.Repository using Postgres.
type GuildRepository struct {
	queries *db.Queries
}

// NewGuildRepository creates a new Postgres guild repository.
func NewGuildRepository(pool *pgxpool.Pool) *GuildRepository {
	return &GuildRepository{
		queries: db.New(pool),
	}
}

// Ensure GuildRepository implements the interface.
var _ appguild.Repository = (*GuildRepository)(nil)

func (r *GuildRepository) Get(ctx context.Context, guildID string) (*guild.Guild, error) {
	row, err := r.queries.GetGuild(ctx, guildID)
	if errors.Is(err, pgx.ErrNoRows) {
		return &guild.Guild{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get guild: %w", err)
	}
	g := &guild.Guild{}
	if row.Language.Valid {
		g.Language = i18n.Parse(row.Language.String)
	}
	if row.WelcomeChannelID.Valid && row.WelcomeMessageID.Valid {
		g.Welcome = &guild.Welcome{ChannelID: row.WelcomeChannelID.String, MessageID: row.WelcomeMessageID.String}
	}
	return g, nil
}

func (r *GuildRepository) SetLanguage(ctx context.Context, guildID string, lang i18n.Lang) error {
	if err := r.queries.SetGuildLanguage(ctx, db.SetGuildLanguageParams{
		GuildID:  guildID,
		Language: pgtype.Text{String: string(lang), Valid: lang != ""},
	}); err != nil {
		return fmt.Errorf("set guild language: %w", err)
	}
	return nil
}

func (r *GuildRepository) SetWelcome(ctx context.Context, guildID string, welcome guild.Welcome) error {
	if err := r.queries.SetGuildWelcome(ctx, db.SetGuildWelcomeParams{
		GuildID:          guildID,
		WelcomeChannelID: pgtype.Text{String: welcome.ChannelID, Valid: true},
		WelcomeMessageID: pgtype.Text{String: welcome.MessageID, Valid: true},
	}); err != nil {
		return fmt.Errorf("set guild welcome: %w", err)
	}
	return nil
}

func (r *GuildRepository) ForgetWelcome(ctx context.Context, guildID string) error {
	if err := r.queries.ForgetGuildWelcome(ctx, guildID); err != nil {
		return fmt.Errorf("forget guild welcome: %w", err)
	}
	return nil
}
