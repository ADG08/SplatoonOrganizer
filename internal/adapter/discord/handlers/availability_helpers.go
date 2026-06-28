package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// refreshWeeklyMessage re-renders the main weekly availability message (best-effort).
func refreshWeeklyMessage(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, week availability.WeekKey, channelID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic while refreshing main table: %v", r)
		}
	}()

	counts, err := svc.GetAvailabilitySummary(ctx, week)
	if err != nil {
		log.Printf("error getting availability summary: %v", err)
		return
	}

	embed := BuildWeeklyEmbed(svc.FormatTable(counts))

	msgID, err := svc.GetSondageMessageID(ctx, week)
	if err != nil {
		log.Printf("error getting sondage message id: %v", err)
		return
	}

	components := BuildWeeklyComponents()
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         msgID,
		Channel:    channelID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	}); err != nil {
		log.Printf("error editing weekly message: %v", err)
	}
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
