package handlers

import (
	"context"
	"log/slog"
	"time"

	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/bwmarrin/discordgo"
)

// GuildMembership pauses the weekly messages of the guilds the bot is removed from, and
// resumes them when the bot is added back: their teams are kept meanwhile. It also has
// the bot introduce itself in the guilds it joins.
type GuildMembership struct {
	teams   *appteam.Service
	welcome *WelcomeHandler
}

// NewGuildMembership creates the guild join and leave handlers.
func NewGuildMembership(teams *appteam.Service, welcome *WelcomeHandler) *GuildMembership {
	return &GuildMembership{teams: teams, welcome: welcome}
}

// OnGuildCreate resumes a guild when the bot joins it, or finds it again on startup.
func (g *GuildMembership) OnGuildCreate(s *discordgo.Session, e *discordgo.GuildCreate) {
	if e.Guild == nil {
		return
	}
	// On startup, Discord announces every guild again: only a recent one is a new join.
	if time.Since(e.JoinedAt) < time.Minute {
		slog.Info("bot added to a guild", "guild", e.ID, "members", e.MemberCount)
		g.welcome.Welcome(s, e.Guild)
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
	g.welcome.Forget(e.ID)
}

// setActive saves whether the guild's teams are active, and reports whether one changed.
func (g *GuildMembership) setActive(guildID string, active bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	changed, err := g.teams.SetGuildActive(ctx, guildID, active)
	if err != nil {
		slog.ErrorContext(ctx, "set guild active", "guild", guildID, "active", active, "err", err)
	}
	return changed
}
