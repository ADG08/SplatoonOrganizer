package handlers

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// SelectDisposHandler handles select menu interactions for day availability.
type SelectDisposHandler struct {
	svc       *appavailability.Service
	teams     *appteam.Service
	refresher *weekly.Refresher
}

// NewSelectDisposHandler creates a new select dispos handler.
func NewSelectDisposHandler(svc *appavailability.Service, teams *appteam.Service, refresher *weekly.Refresher) *SelectDisposHandler {
	return &SelectDisposHandler{svc: svc, teams: teams, refresher: refresher}
}

func (h *SelectDisposHandler) CustomID() string {
	return SelectDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*SelectDisposHandler)(nil)

func (h *SelectDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	t, day := loadTeam(ctx, s, i, h.teams, SelectDisposCustomIDPrefix)
	if t == nil {
		return nil
	}
	m, layout := t.Messages(), t.SlotLayout
	userID := getUserID(i)
	if userID == "" {
		slog.WarnContext(ctx, "could not determine user ID", "guild", i.GuildID)
		return discord.RespondError(s, i, m.ErrUnknownUser)
	}
	dayIndex, err := parseDayIndex(day)
	if err != nil {
		slog.WarnContext(ctx, "invalid select custom id", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrInvalid)
	}

	week := h.svc.Week(t.Schedule, t.Location)
	userSlots, err := h.svc.UserAvailability(ctx, t.ID, week, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get user availability", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	// Values are 1-based slot indexes of the layout ("1" = first slot). Closed and past
	// slots aren't offered, and are refused if sent anyway (from menus opened earlier).
	// The past ones keep what the member had picked: the day is saved as a whole.
	closed := availability.ClosedSlots(layout, t.Schedule, week, t.Location)
	past := availability.PastSlots(layout, week, t.Location, time.Now())
	slots := make([]bool, layout.Len())
	for slot := range slots {
		slots[slot] = past[dayIndex][slot] && userSlots[dayIndex][slot]
	}
	for _, v := range i.MessageComponentData().Values {
		n, err := strconv.Atoi(v)
		if err != nil || !layout.Valid(n-1) || closed[dayIndex][n-1] || past[dayIndex][n-1] {
			slog.WarnContext(ctx, "ignoring invalid slot value", "guild", i.GuildID, "value", v)
			continue
		}
		slots[n-1] = true
	}

	if err := h.svc.SetDayAvailability(ctx, t.ID, week, userID, dayIndex, slots); err != nil {
		slog.ErrorContext(ctx, "set day availability", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	if userSlots[dayIndex] == nil {
		userSlots[dayIndex] = make(map[int]bool)
	}
	for slot, picked := range slots {
		userSlots[dayIndex][slot] = picked
	}
	var loc *time.Location // the team's time, unless members see their own
	if t.MemberTime() {
		if loc, err = h.svc.UserTimezone(ctx, userID); err != nil {
			slog.ErrorContext(ctx, "get user timezone after select", "guild", i.GuildID, "err", err)
		}
	}
	label := func(day, slot int) string { return h.svc.SlotLabelIn(m, layout, t.Location, week, day, slot, loc) }

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last
	// days, the latter followed by the "change my timezone" button for hourly slots).
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(m, t.ID, layout, userSlots, closed, past, 0, 5, label)
	} else {
		components = buildUserDisposSelectMenus(m, t.ID, layout, userSlots, closed, past, 5, 2, label)
		if t.MemberTime() {
			components = append(components, editTimezoneButton(m, t.ID, tzNextDispos))
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    withLegend(withTeam(t, m.DisposUpdated), timezoneLegend(m, layout, loc)),
			Components: components,
		},
	}); err != nil {
		slog.ErrorContext(ctx, "respond to select", "guild", i.GuildID, "err", err)
	}

	// Refresh main weekly table (best-effort)
	h.refresher.Request(s, t.ID)

	return nil
}
