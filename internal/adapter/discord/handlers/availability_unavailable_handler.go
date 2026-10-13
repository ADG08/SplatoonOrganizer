package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/bwmarrin/discordgo"
)

// UnavailableDisposHandler handles the "Pas dispo cette semaine" button click.
type UnavailableDisposHandler struct {
	svc       *appavailability.Service
	refresher *WeeklyRefresher
}

// NewUnavailableDisposHandler creates a new unavailable dispos handler.
func NewUnavailableDisposHandler(svc *appavailability.Service, refresher *WeeklyRefresher) *UnavailableDisposHandler {
	return &UnavailableDisposHandler{svc: svc, refresher: refresher}
}

func (h *UnavailableDisposHandler) CustomID() string {
	return UnavailableCustomIDPrefix
}

var _ discord.InteractionHandler = (*UnavailableDisposHandler)(nil)

func (h *UnavailableDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		log.Printf("unavailable dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	ctx := context.Background()

	nowUnavailable, err := h.svc.ToggleWeekUnavailable(ctx, i.GuildID, userID)
	if err != nil {
		log.Printf("error toggling week unavailability: %v", err)
		return respondWithError(s, i, "Erreur lors de la mise à jour de ta disponibilité.")
	}

	content := "✅ Tu es de nouveau disponible cette semaine. Configure tes créneaux avec **✏️ Mes dispos**."
	if nowUnavailable {
		content = "🚫 Tu es marqué **indisponible** pour toute la semaine. Tes créneaux ont été effacés. Reclique sur le bouton (ou coche un créneau) pour revenir disponible."
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
