package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/bwmarrin/discordgo"
)

// OpenDisposHandler handles the "Mes dispos" button click.
type OpenDisposHandler struct {
	svc *appavailability.Service
}

// NewOpenDisposHandler creates a new open dispos handler.
func NewOpenDisposHandler(svc *appavailability.Service) *OpenDisposHandler {
	return &OpenDisposHandler{svc: svc}
}

func (h *OpenDisposHandler) CustomID() string {
	return OpenDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*OpenDisposHandler)(nil)

func (h *OpenDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		log.Printf("open dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	ctx := context.Background()

	_, userSlots, err := h.svc.GetUserAvailability(ctx, userID)
	if err != nil {
		log.Printf("error getting user availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de vos disponibilités.")
	}

	// Discord: max 5 Action Rows per message, 1 select menu per row -> 5 select menus per message.
	// We have 7 days: first message 5 menus (Lundi-Vendredi), follow-up 2 menus (Samedi-Dimanche).
	componentsFirst := buildUserDisposSelectMenus(userSlots, 0, 5)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Configure tes disponibilités pour la semaine en cours :",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: componentsFirst,
		},
	}); err != nil {
		return err
	}
	componentsSecond := buildUserDisposSelectMenus(userSlots, 5, 2)
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content:    "Suite :",
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: componentsSecond,
	})
	return err
}

// ViewDisposHandler handles the "Voir les dispos" button click.
type ViewDisposHandler struct {
	svc *appavailability.Service
}

// NewViewDisposHandler creates a new view dispos handler.
func NewViewDisposHandler(svc *appavailability.Service) *ViewDisposHandler {
	return &ViewDisposHandler{svc: svc}
}

func (h *ViewDisposHandler) CustomID() string {
	return ViewDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*ViewDisposHandler)(nil)

func (h *ViewDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx := context.Background()
	week := h.svc.CurrentWeek()

	users, err := h.svc.GetAvailabilityUsers(ctx, week)
	if err != nil {
		log.Printf("error getting availability users from button: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération des disponibilités.")
	}

	description := h.svc.FormatUsersBySlot(users)
	embed := &discordgo.MessageEmbed{
		Title:       "Disponibilités par créneau",
		Description: description,
		Color:       0xFEE75C,
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
