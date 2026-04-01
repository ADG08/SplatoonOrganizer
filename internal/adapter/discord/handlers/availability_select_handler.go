package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// SelectDisposHandler handles select menu interactions for day availability.
type SelectDisposHandler struct {
	svc            *appavailability.Service
	defaultChannel string
}

// NewSelectDisposHandler creates a new select dispos handler.
func NewSelectDisposHandler(svc *appavailability.Service, defaultChannelID string) *SelectDisposHandler {
	return &SelectDisposHandler{
		svc:            svc,
		defaultChannel: defaultChannelID,
	}
}

func (h *SelectDisposHandler) CustomID() string {
	return SelectDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*SelectDisposHandler)(nil)

func (h *SelectDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		log.Printf("select dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	dayIndex, err := parseSelectDisposCustomID(i.MessageComponentData().CustomID)
	if err != nil {
		log.Printf("invalid select custom id: %v", err)
		return respondWithError(s, i, "Interaction invalide.")
	}

	ctx := context.Background()

	// Values are "1" and "2" for Après-midi and Soir
	values := i.MessageComponentData().Values
	var slots [availability.SlotsPerDay]bool
	for _, v := range values {
		switch v {
		case "1":
			slots[0] = true
		case "2":
			slots[1] = true
		}
	}

	week, err := h.svc.SetDayAvailability(ctx, userID, dayIndex, slots)
	if err != nil {
		log.Printf("error setting day availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la mise à jour de vos disponibilités.")
	}

	_, userSlots, err := h.svc.GetUserAvailability(ctx, userID)
	if err != nil {
		log.Printf("error getting user availability after select: %v", err)
		return respondWithError(s, i, "Erreur lors du rafraîchissement.")
	}

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last days)
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(userSlots, 0, 5)
	} else {
		components = buildUserDisposSelectMenus(userSlots, 5, 2)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Tes disponibilités ont été mises à jour.",
			Components: components,
		},
	}); err != nil {
		log.Printf("error responding to select: %v", err)
	}

	// Refresh main weekly table (best-effort)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic while refreshing main table: %v", r)
			}
		}()

		counts, err := h.svc.GetAvailabilitySummary(ctx, week)
		if err != nil {
			log.Printf("error getting availability summary: %v", err)
			return
		}

		content := h.svc.FormatTable(counts)
		embed := BuildWeeklyEmbed(content)

		msgID, err := h.svc.GetSondageMessageID(ctx, week)
		if err != nil {
			log.Printf("error getting sondage message id: %v", err)
			return
		}

		channelID := h.defaultChannel
		if i.ChannelID != "" {
			channelID = i.ChannelID
		}
		components := BuildWeeklyComponents()
		_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         msgID,
			Channel:    channelID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err != nil {
			log.Printf("error editing weekly message: %v", err)
		}
	}()

	return nil
}
