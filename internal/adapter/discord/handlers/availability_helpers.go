package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

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
