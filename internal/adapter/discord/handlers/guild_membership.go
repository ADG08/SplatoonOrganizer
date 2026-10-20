package handlers

import (
	"context"
	"log/slog"
	"time"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// GuildMembership pauses the weekly message of the guilds the bot is removed from, and
// resumes it when the bot is added back: their configuration is kept meanwhile.
type GuildMembership struct {
	guildCfg *appguildconfig.Service
}

// NewGuildMembership creates the guild join and leave handlers.
func NewGuildMembership(guildCfg *appguildconfig.Service) *GuildMembership {
	return &GuildMembership{guildCfg: guildCfg}
}

// OnGuildCreate resumes a guild when the bot joins it, or finds it again on startup.
func (g *GuildMembership) OnGuildCreate(_ *discordgo.Session, e *discordgo.GuildCreate) {
	if e.Guild == nil {
		return
	}
	// On startup, Discord announces every guild again: only a recent one is a new join.
	if time.Since(e.JoinedAt) < time.Minute {
		slog.Info("bot added to a guild", "guild", e.ID, "members", e.MemberCount)
	}
	if g.setActive(e.ID, true) {
		slog.Info("guild resumed", "guild", e.ID)
	}
}

// OnGuildDelete pauses a guild the bot was removed from. A Discord outage also sends it,
// marked unavailable: that one is not a removal.
func (g *GuildMembership) OnGuildDelete(_ *discordgo.Session, e *discordgo.GuildDelete) {
	if e.Guild == nil || e.Unavailable {
		return
	}
	var name string // the guild is no longer known by its ID
	if e.BeforeDelete != nil {
		name = e.BeforeDelete.Name
	}
	slog.Info("bot removed from a guild, guild paused", "guild", e.ID, "name", name)
	g.setActive(e.ID, false)
}

// setActive saves whether the guild is active, and reports whether that changed.
func (g *GuildMembership) setActive(guildID string, active bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// On startup, every guild is announced: only write the ones that change.
	cfg, err := g.guildCfg.Get(ctx, guildID)
	if err != nil {
		slog.ErrorContext(ctx, "get guild config", "guild", guildID, "err", err)
		return false
	}
	if cfg.Active == active {
		return false
	}
	if err := g.guildCfg.SetActive(ctx, guildID, active); err != nil {
		slog.ErrorContext(ctx, "set guild active", "guild", guildID, "active", active, "err", err)
		return false
	}
	return true
}
