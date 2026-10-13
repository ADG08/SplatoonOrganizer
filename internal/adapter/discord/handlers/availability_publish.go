package handlers

import (
	"context"
	"fmt"
	"log"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// PublishWeeklyMessage replaces the weekly message: it deletes the known weekly messages
// of the channel, then posts a fresh one with the guild's layout and display.
// Used by the weekly job and when /help changes how the message looks (an embed can't
// always be edited into a Components V2 message, so it is posted again instead).
func PublishWeeklyMessage(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig, channelID string) error {
	week := svc.CurrentWeek()
	counts, err := svc.GetAvailabilitySummary(ctx, week)
	if err != nil {
		return fmt.Errorf("get availability summary: %w", err)
	}

	payload, err := BuildWeeklyPayload(svc.BuildWeeklyView(cfg.SlotLayout, week, counts), cfg.RoleID, cfg.Display)
	if err != nil {
		return err
	}

	ids, err := svc.ListSondageMessageIDs(ctx)
	if err != nil {
		log.Printf("list sondage message ids: %v", err)
	} else {
		for _, msgID := range ids {
			if msgID == "" {
				continue
			}
			if err := s.ChannelMessageDelete(channelID, msgID); err != nil {
				log.Printf("error deleting sondage message %s: %v", msgID, err)
			}
		}
	}

	msg, err := s.ChannelMessageSendComplex(channelID, payload.MessageSend())
	if err != nil {
		return fmt.Errorf("send weekly message: %w", err)
	}
	if err := svc.UpsertSondageMessage(ctx, week, msg.ID); err != nil {
		return fmt.Errorf("upsert sondage message: %w", err)
	}
	return nil
}
