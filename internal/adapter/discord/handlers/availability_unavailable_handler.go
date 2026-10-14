package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// UnavailableDisposHandler handles the "Pas dispo cette semaine" button click.
type UnavailableDisposHandler struct {
	svc       *appavailability.Service
	guildCfg  *appguildconfig.Service
	refresher *weekly.Refresher
}

// NewUnavailableDisposHandler creates a new unavailable dispos handler.
func NewUnavailableDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service, refresher *weekly.Refresher) *UnavailableDisposHandler {
	return &UnavailableDisposHandler{svc: svc, guildCfg: guildCfg, refresher: refresher}
}

func (h *UnavailableDisposHandler) CustomID() string {
	return weekly.UnavailableCustomID
}

var _ discord.InteractionHandler = (*UnavailableDisposHandler)(nil)

func (h *UnavailableDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	userID := getUserID(i)
	if userID == "" {
		log.Printf("unavailable dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, m.ErrUnknownUser)
	}

	nowUnavailable, err := h.svc.ToggleWeekUnavailable(ctx, i.GuildID, h.svc.Week(cfg.Schedule), userID)
	if err != nil {
		log.Printf("error toggling week unavailability: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	content := m.AvailableAgain
	if nowUnavailable {
		content = m.UnavailableNow
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("error responding to unavailable toggle: %v", err)
	}

	// Marking unavailable clears slots, so refresh the main weekly table (best-effort).
	h.refresher.Request(s, i.GuildID)

	return nil
}
