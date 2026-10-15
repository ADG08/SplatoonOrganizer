package handlers

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// SelectDisposHandler handles select menu interactions for day availability.
type SelectDisposHandler struct {
	svc       *appavailability.Service
	guildCfg  *appguildconfig.Service
	refresher *weekly.Refresher
}

// NewSelectDisposHandler creates a new select dispos handler.
func NewSelectDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service, refresher *weekly.Refresher) *SelectDisposHandler {
	return &SelectDisposHandler{svc: svc, guildCfg: guildCfg, refresher: refresher}
}

func (h *SelectDisposHandler) CustomID() string {
	return SelectDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*SelectDisposHandler)(nil)

func (h *SelectDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m, layout := cfg.Messages(), cfg.SlotLayout
	userID := getUserID(i)
	if userID == "" {
		slog.WarnContext(ctx, "could not determine user ID", "guild", i.GuildID)
		return respondWithError(s, i, m.ErrUnknownUser)
	}
	dayIndex, err := parseSelectDisposCustomID(i.MessageComponentData().CustomID)
	if err != nil {
		slog.WarnContext(ctx, "invalid select custom id", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrInvalid)
	}

	// Values are 1-based slot indexes of the layout ("1" = first slot). Closed slots aren't
	// offered, and are refused if sent anyway.
	week := h.svc.Week(cfg.Schedule, cfg.Location)
	closed := availability.ClosedSlots(layout, cfg.Schedule, week, cfg.Location)
	slots := make([]bool, layout.Len())
	for _, v := range i.MessageComponentData().Values {
		n, err := strconv.Atoi(v)
		if err != nil || !layout.Valid(n-1) || closed[dayIndex][n-1] {
			slog.WarnContext(ctx, "ignoring invalid slot value", "guild", i.GuildID, "value", v)
			continue
		}
		slots[n-1] = true
	}

	if err := h.svc.SetDayAvailability(ctx, i.GuildID, week, userID, dayIndex, slots); err != nil {
		slog.ErrorContext(ctx, "set day availability", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	userSlots, err := h.svc.UserAvailability(ctx, i.GuildID, week, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get user availability after select", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	var loc *time.Location // the guild's time, unless members see their own
	if cfg.MemberTime() {
		if loc, err = h.svc.UserTimezone(ctx, userID); err != nil {
			slog.ErrorContext(ctx, "get user timezone after select", "guild", i.GuildID, "err", err)
		}
	}
	label := func(day, slot int) string { return h.svc.SlotLabelIn(m, layout, cfg.Location, week, day, slot, loc) }

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last
	// days, the latter followed by the "change my timezone" button for hourly slots).
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(m, layout, userSlots, closed, 0, 5, label)
	} else {
		components = buildUserDisposSelectMenus(m, layout, userSlots, closed, 5, 2, label)
		if cfg.MemberTime() {
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
		slog.ErrorContext(ctx, "respond to select", "guild", i.GuildID, "err", err)
	}

	// Refresh main weekly table (best-effort)
	h.refresher.Request(s, i.GuildID)

	return nil
}
