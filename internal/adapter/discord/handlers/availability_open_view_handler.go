package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
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
	return weekly.OpenDisposCustomID
}

var _ discord.InteractionHandler = (*OpenDisposHandler)(nil)

func (h *OpenDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	userID := getUserID(i)
	if userID == "" {
		slog.WarnContext(ctx, "could not determine user ID", "guild", i.GuildID)
		return respondWithError(s, i, m.ErrUnknownUser)
	}

	// Opening "Mes dispos" means the user is available again: clear any week-unavailable status.
	if err := h.svc.ClearWeekUnavailable(ctx, i.GuildID, h.svc.Week(cfg.Schedule, cfg.Location), userID); err != nil {
		slog.ErrorContext(ctx, "clear week unavailability on open", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	// Hourly slots are shown in the member's timezone: ask for it the first time.
	var loc *time.Location
	if cfg.SlotLayout.IsHourly() {
		var err error
		if loc, err = h.svc.UserTimezone(ctx, userID); err != nil {
			slog.ErrorContext(ctx, "get user timezone", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		if loc == nil {
			return respondTimezonePicker(s, i, m, tzNextDispos)
		}
	}
	return sendDisposMenus(ctx, s, i, h.svc, cfg, userID, loc, false)
}

// sendDisposMenus sends the "Mes dispos" menus (5 days, then 2 days because of Discord's
// 5 rows limit), labelled in the member's timezone. With followup, both are follow-ups.
func sendDisposMenus(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, cfg *guildconfig.GuildConfig, userID string, loc *time.Location, followup bool) error {
	m, layout := cfg.Messages(), cfg.SlotLayout
	week := svc.Week(cfg.Schedule, cfg.Location)
	userSlots, err := svc.UserAvailability(ctx, i.GuildID, week, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get user availability", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	label := func(day, slot int) string { return svc.SlotLabelIn(m, layout, cfg.Location, week, day, slot, loc) }
	closed := availability.ClosedSlots(layout, cfg.Schedule, week, cfg.Location)

	first := &discordgo.WebhookParams{
		Content:    withLegend(m.DisposIntro, timezoneLegend(m, layout, loc)),
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: buildUserDisposSelectMenus(m, layout, userSlots, closed, 0, 5, label),
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
	second := buildUserDisposSelectMenus(m, layout, userSlots, closed, 5, 2, label)
	if layout.IsHourly() {
		second = append(second, editTimezoneButton(m, tzNextDispos))
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
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewViewDisposHandler creates a new view dispos handler.
func NewViewDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *ViewDisposHandler {
	return &ViewDisposHandler{svc: svc, guildCfg: guildCfg}
}

func (h *ViewDisposHandler) CustomID() string {
	return weekly.ViewDisposCustomID
}

var _ discord.InteractionHandler = (*ViewDisposHandler)(nil)

func (h *ViewDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	week := h.svc.Week(cfg.Schedule, cfg.Location)

	users, err := h.svc.AvailabilityUsers(ctx, i.GuildID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability users from button", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	unavailableUsers, err := h.svc.WeekUnavailableUsers(ctx, i.GuildID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get week unavailable users from button", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	description := h.svc.FormatUsersBySlot(m, cfg.SlotLayout, cfg.Location, week, users)
	if unavailable := h.svc.FormatUnavailableUsers(m, unavailableUsers); unavailable != "" {
		description += "\n" + unavailable
	}

	// Hourly details can exceed the embed description limit: split into several messages.
	chunks := splitLines(description, embedDescriptionLimit)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       m.ViewTitle,
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
