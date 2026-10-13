package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// SelectDisposHandler handles select menu interactions for day availability.
type SelectDisposHandler struct {
	svc       *appavailability.Service
	guildCfg  *appguildconfig.Service
	refresher *WeeklyRefresher
}

// NewSelectDisposHandler creates a new select dispos handler.
func NewSelectDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service, refresher *WeeklyRefresher) *SelectDisposHandler {
	return &SelectDisposHandler{svc: svc, guildCfg: guildCfg, refresher: refresher}
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

	// Values are 1-based slot indexes of the layout ("1" = first slot)
	layout, err := h.guildCfg.SlotLayout(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting slot layout: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration du serveur.")
	}
	slots := make([]bool, layout.Len())
	for _, v := range i.MessageComponentData().Values {
		n, err := strconv.Atoi(v)
		if err != nil || !layout.Valid(n-1) {
			log.Printf("ignoring invalid slot value %q", v)
			continue
		}
		slots[n-1] = true
	}

	week, err := h.svc.SetDayAvailability(ctx, i.GuildID, userID, dayIndex, slots)
	if err != nil {
		log.Printf("error setting day availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la mise à jour de vos disponibilités.")
	}

	_, userSlots, err := h.svc.GetUserAvailability(ctx, i.GuildID, userID)
	if err != nil {
		log.Printf("error getting user availability after select: %v", err)
		return respondWithError(s, i, "Erreur lors du rafraîchissement.")
	}
	loc, err := h.svc.UserTimezone(ctx, userID)
	if err != nil {
		log.Printf("error getting user timezone after select: %v", err)
	}
	label := func(day, slot int) string { return h.svc.SlotLabelIn(layout, week, day, slot, loc) }

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last days)
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(layout, userSlots, 0, 5, label)
	} else {
		components = buildUserDisposSelectMenus(layout, userSlots, 5, 2, label)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    withLegend("Tes disponibilités ont été mises à jour.", timezoneLegend(layout, loc)),
			Components: components,
		},
	}); err != nil {
		log.Printf("error responding to select: %v", err)
	}

	// Refresh main weekly table (best-effort)
	h.refresher.Request(s, i.GuildID)

	return nil
}
