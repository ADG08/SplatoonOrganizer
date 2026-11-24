package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/bwmarrin/discordgo"
)

// embedDescriptionLimit is Discord's maximum length of an embed description.
const embedDescriptionLimit = 4096

// splitLines splits text into chunks of at most limit bytes, cutting between lines when possible.
func splitLines(text string, limit int) []string {
	var chunks []string
	var cur strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		for len(line) > limit { // a single line longer than the limit: hard cut on a rune boundary
			if cur.Len() > 0 {
				chunks = append(chunks, cur.String())
				cur.Reset()
			}
			cut := limit
			for cut > 0 && !utf8.RuneStart(line[cut]) {
				cut--
			}
			chunks = append(chunks, line[:cut])
			line = line[cut:]
		}
		if cur.Len()+len(line) > limit {
			chunks = append(chunks, cur.String())
			cur.Reset()
		}
		cur.WriteString(line)
	}
	if cur.Len() > 0 || len(chunks) == 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}

// withLegend appends the optional timezone legend to an ephemeral message content.
func withLegend(content, legend string) string {
	if legend == "" {
		return content
	}
	return content + "\n" + legend
}

// withTeam says which team an ephemeral message is about: a member can be in several.
func withTeam(t *team.Team, content string) string {
	return "**" + t.Name + "** · " + content
}

// loadTeam returns the team of a weekly message's component, from its custom ID, and what
// follows the team's ID there. The components of a message posted before teams existed
// have no ID: they belong to the guild's oldest team, which took over its configuration.
// When the team can't be read or is gone, it answers the member and returns nil.
func loadTeam(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, teams *appteam.Service, prefix string) (*team.Team, string) {
	m := discord.LocaleMessages(i)
	teamID, rest, err := weekly.ParseCustomID(i.MessageComponentData().CustomID, prefix)
	if err != nil {
		slog.WarnContext(ctx, "invalid custom id", "guild", i.GuildID, "err", err)
		if err := discord.RespondError(s, i, m.ErrInvalid); err != nil {
			slog.ErrorContext(ctx, "respond", "guild", i.GuildID, "err", err)
		}
		return nil, ""
	}

	var t *team.Team
	if teamID == 0 {
		t, err = teams.Primary(ctx, i.GuildID)
	} else {
		t, err = teams.GetInGuild(ctx, i.GuildID, teamID)
	}
	if err == nil {
		return t, rest
	}

	text := m.ErrGeneric
	if errors.Is(err, team.ErrNotFound) {
		text = m.TeamGone
	} else {
		slog.ErrorContext(ctx, "get team", "guild", i.GuildID, "err", err)
	}
	if err := discord.RespondError(s, i, text); err != nil {
		slog.ErrorContext(ctx, "respond", "guild", i.GuildID, "err", err)
	}
	return nil, ""
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// parseDayIndex reads the day of a "Mes dispos" menu, the end of its custom ID.
func parseDayIndex(s string) (int, error) {
	day, err := strconv.Atoi(s)
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return 0, fmt.Errorf("invalid day index: %q", s)
	}
	return day, nil
}
