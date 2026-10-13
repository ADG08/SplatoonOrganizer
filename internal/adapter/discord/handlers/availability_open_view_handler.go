package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// OpenDisposHandler handles the "Mes dispos" button click.
type OpenDisposHandler struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewOpenDisposHandler creates a new open dispos handler.
func NewOpenDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *OpenDisposHandler {
	return &OpenDisposHandler{svc: svc, guildCfg: guildCfg}
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

	// Opening "Mes dispos" means the user is available again: clear any week-unavailable status.
	if err := h.svc.ClearWeekUnavailable(ctx, userID); err != nil {
		log.Printf("error clearing week unavailability on open: %v", err)
		return respondWithError(s, i, "Erreur lors de la mise à jour de vos disponibilités.")
	}

	layout, err := h.guildCfg.SlotLayout(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting slot layout: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration du serveur.")
	}
	return sendDisposMenus(s, i, h.svc, layout, userID)
}

// sendDisposMenus sends the "Mes dispos" menus (5 days, then 2 days because of Discord's
// 5 rows limit).
func sendDisposMenus(s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, layout availability.Layout, userID string) error {
	_, userSlots, err := svc.GetUserAvailability(context.Background(), userID)
	if err != nil {
		log.Printf("error getting user availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de vos disponibilités.")
	}
	label := func(day, slot int) string { return layout.Label(slot) }

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Configure tes disponibilités pour la semaine en cours :",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: buildUserDisposSelectMenus(layout, userSlots, 0, 5, label),
		},
	}); err != nil {
		return err
	}
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content:    "Suite :",
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: buildUserDisposSelectMenus(layout, userSlots, 5, 2, label),
	})
	return err
}

// ViewDisposHandler handles the "Voir les dispos" button click.
type ViewDisposHandler struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewViewDisposHandler creates a new view dispos handler.
func NewViewDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *ViewDisposHandler {
	return &ViewDisposHandler{svc: svc, guildCfg: guildCfg}
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

	unavailableUsers, err := h.svc.GetWeekUnavailableUsers(ctx, week)
	if err != nil {
		log.Printf("error getting week unavailable users from button: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération des disponibilités.")
	}

	layout, err := h.guildCfg.SlotLayout(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting slot layout: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration du serveur.")
	}

	description := h.svc.FormatUsersBySlot(layout, week, users)
	if unavailable := h.svc.FormatUnavailableUsers(unavailableUsers); unavailable != "" {
		description += "\n" + unavailable
	}

	// Hourly details can exceed the embed description limit: split into several messages.
	chunks := splitLines(description, embedDescriptionLimit)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Disponibilités par créneau",
				Description: chunks[0],
				Color:       0xFEE75C,
			}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}
	for _, chunk := range chunks[1:] {
		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{Description: chunk, Color: 0xFEE75C}},
			Flags:  discordgo.MessageFlagsEphemeral,
		}); err != nil {
			return err
		}
	}
	return nil
}
