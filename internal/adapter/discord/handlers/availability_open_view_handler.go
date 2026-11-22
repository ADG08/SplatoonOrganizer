package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/bwmarrin/discordgo"
)

// OpenDisposHandler handles the "Mes dispos" button click.
type OpenDisposHandler struct {
	svc   *appavailability.Service
	teams *appteam.Service
}

// NewOpenDisposHandler creates a new open dispos handler.
func NewOpenDisposHandler(svc *appavailability.Service, teams *appteam.Service) *OpenDisposHandler {
	return &OpenDisposHandler{svc: svc, teams: teams}
}

func (h *OpenDisposHandler) CustomID() string {
	return weekly.OpenDisposCustomID
}

var _ discord.InteractionHandler = (*OpenDisposHandler)(nil)

func (h *OpenDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	t, _ := loadTeam(ctx, s, i, h.teams, weekly.OpenDisposCustomID)
	if t == nil {
		return nil
	}
	m := t.Messages()
	userID := getUserID(i)
	if userID == "" {
		slog.WarnContext(ctx, "could not determine user ID", "guild", i.GuildID)
		return discord.RespondError(s, i, m.ErrUnknownUser)
	}

	// Opening "Mes dispos" means the user is available again: clear any week-unavailable status.
	if err := h.svc.ClearWeekUnavailable(ctx, t.ID, h.svc.Week(t.Schedule, t.Location), userID); err != nil {
		slog.ErrorContext(ctx, "clear week unavailability on open", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	// In an international team, hourly slots are shown in the member's timezone: ask for
	// it the first time.
	var loc *time.Location
	if t.MemberTime() {
		var err error
		if loc, err = h.svc.UserTimezone(ctx, userID); err != nil {
			slog.ErrorContext(ctx, "get user timezone", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
		if loc == nil {
			return respondTimezonePicker(s, i, t, tzNextDispos)
		}
	}
	return sendDisposMenus(ctx, s, i, h.svc, t, userID, loc, false)
}

// sendDisposMenus sends the "Mes dispos" menus (5 days, then 2 days because of Discord's
// 5 rows limit), labelled in the member's timezone loc when the team shows member time,
// else in the team's. With followup, both are follow-ups.
func sendDisposMenus(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, t *team.Team, userID string, loc *time.Location, followup bool) error {
	m, layout := t.Messages(), t.SlotLayout
	if !t.MemberTime() {
		loc = nil // a picker left from when the team showed member time
	}
	week := svc.Week(t.Schedule, t.Location)
	userSlots, err := svc.UserAvailability(ctx, t.ID, week, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get user availability", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	label := func(day, slot int) string { return svc.SlotLabelIn(m, layout, t.Location, week, day, slot, loc) }
	closed := availability.ClosedSlots(layout, t.Schedule, week, t.Location)
	past := availability.PastSlots(layout, week, t.Location, time.Now())

	first := &discordgo.WebhookParams{
		Content:    withLegend(withTeam(t, m.DisposIntro), timezoneLegend(m, layout, loc)),
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: buildUserDisposSelectMenus(m, t.ID, layout, userSlots, closed, past, 0, 5, label),
	}
	if followup {
		if _, err := s.FollowupMessageCreate(i.Interaction, true, first); err != nil {
			return err
		}
	} else if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: first.Content, Flags: first.Flags, Components: first.Components},
	}); err != nil {
		return err
	}
	second := buildUserDisposSelectMenus(m, t.ID, layout, userSlots, closed, past, 5, 2, label)
	if t.MemberTime() {
		second = append(second, editTimezoneButton(m, t.ID, tzNextDispos))
	}
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content:    m.DisposMore,
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: second,
	})
	return err
}

// ViewDisposHandler handles the "Voir les dispos" button click.
type ViewDisposHandler struct {
	svc   *appavailability.Service
	teams *appteam.Service
}

// NewViewDisposHandler creates a new view dispos handler.
func NewViewDisposHandler(svc *appavailability.Service, teams *appteam.Service) *ViewDisposHandler {
	return &ViewDisposHandler{svc: svc, teams: teams}
}

func (h *ViewDisposHandler) CustomID() string {
	return weekly.ViewDisposCustomID
}

var _ discord.InteractionHandler = (*ViewDisposHandler)(nil)

func (h *ViewDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	t, _ := loadTeam(ctx, s, i, h.teams, weekly.ViewDisposCustomID)
	if t == nil {
		return nil
	}
	return h.show(ctx, s, i, t)
}

// show answers with who is available on each slot of the team's week.
func (h *ViewDisposHandler) show(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, t *team.Team) error {
	m := t.Messages()
	week := h.svc.Week(t.Schedule, t.Location)

	users, err := h.svc.AvailabilityUsers(ctx, t.ID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability users from button", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	unavailableUsers, err := h.svc.WeekUnavailableUsers(ctx, t.ID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get week unavailable users from button", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	description := h.svc.FormatUsersBySlot(m, t.SlotLayout, t.Location, week, users)
	if unavailable := h.svc.FormatUnavailableUsers(m, unavailableUsers); unavailable != "" {
		description += "\n" + unavailable
	}

	// Hourly details can exceed the embed description limit: split into several messages.
	chunks := splitLines(description, embedDescriptionLimit)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       weekly.Title(t, m.ViewTitle),
				Description: chunks[0],
				Color:       weekly.AccentColor,
			}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}
	for _, chunk := range chunks[1:] {
		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{Description: chunk, Color: weekly.AccentColor}},
			Flags:  discordgo.MessageFlagsEphemeral,
		}); err != nil {
			return err
		}
	}
	return nil
}
