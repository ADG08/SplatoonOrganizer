package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
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

// loadGuildConfig returns the config (and so the language) of the interaction's guild.
// When it can't be read, it answers the member in French and returns nil.
func loadGuildConfig(s *discordgo.Session, i *discordgo.InteractionCreate, guildCfg *appguildconfig.Service) *guildconfig.GuildConfig {
	cfg, err := guildCfg.Get(context.Background(), i.GuildID)
	if err != nil {
		log.Printf("guild %s: get guild config: %v", i.GuildID, err)
		if err := respondWithError(s, i, i18n.For(i18n.FR).ErrGeneric); err != nil {
			log.Printf("respond: %v", err)
		}
		return nil
	}
	return cfg
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func parseSelectDisposCustomID(customID string) (int, error) {
	if !strings.HasPrefix(customID, SelectDisposCustomIDPrefix+":") {
		return 0, fmt.Errorf("invalid custom id: %s", customID)
	}
	s := customID[len(SelectDisposCustomIDPrefix)+1:]
	day, err := strconv.Atoi(s)
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return 0, fmt.Errorf("invalid day index: %s", s)
	}
	return day, nil
}
