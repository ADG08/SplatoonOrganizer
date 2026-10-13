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
	cfg := loadGuildConfig(s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m, layout := cfg.Messages(), cfg.SlotLayout
	userID := getUserID(i)
	if userID == "" {
		log.Printf("select dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, m.ErrUnknownUser)
	}
	dayIndex, err := parseSelectDisposCustomID(i.MessageComponentData().CustomID)
	if err != nil {
		log.Printf("invalid select custom id: %v", err)
		return respondWithError(s, i, m.ErrInvalid)
	}

	ctx := context.Background()

	// Values are 1-based slot indexes of the layout ("1" = first slot)
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
		return respondWithError(s, i, m.ErrGeneric)
	}

	_, userSlots, err := h.svc.GetUserAvailability(ctx, i.GuildID, userID)
	if err != nil {
		log.Printf("error getting user availability after select: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	loc, err := h.svc.UserTimezone(ctx, userID)
	if err != nil {
		log.Printf("error getting user timezone after select: %v", err)
	}
	label := func(day, slot int) string { return h.svc.SlotLabelIn(m, layout, week, day, slot, loc) }

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last
	// days, the latter followed by the "change my timezone" button for hourly slots).
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(m, layout, userSlots, 0, 5, label)
	} else {
		components = buildUserDisposSelectMenus(m, layout, userSlots, 5, 2, label)
		if layout.IsHourly() {
			components = append(components, editTimezoneButton(m, tzNextDispos))
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    withLegend(m.DisposUpdated, timezoneLegend(m, layout, loc)),
			Components: components,
		},
	}); err != nil {
		log.Printf("error responding to select: %v", err)
	}

	// Refresh main weekly table (best-effort)
	h.refresher.Request(s, i.GuildID)

	return nil
}
