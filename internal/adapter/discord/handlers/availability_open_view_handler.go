package handlers

import (
	"context"
	"log"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
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
	return OpenDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*OpenDisposHandler)(nil)

func (h *OpenDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	userID := getUserID(i)
	if userID == "" {
		log.Printf("open dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, m.ErrUnknownUser)
	}
	ctx := context.Background()

	// Opening "Mes dispos" means the user is available again: clear any week-unavailable status.
	if err := h.svc.ClearWeekUnavailable(ctx, i.GuildID, h.svc.Week(cfg.Schedule), userID); err != nil {
		log.Printf("error clearing week unavailability on open: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	// Hourly slots are shown in the member's timezone: ask for it the first time.
	var loc *time.Location
	if cfg.SlotLayout.IsHourly() {
		var err error
		if loc, err = h.svc.UserTimezone(ctx, userID); err != nil {
			log.Printf("error getting user timezone: %v", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		if loc == nil {
			return respondTimezonePicker(s, i, m, tzNextDispos)
		}
	}
	return sendDisposMenus(s, i, h.svc, cfg, userID, loc, false)
}

// sendDisposMenus sends the "Mes dispos" menus (5 days, then 2 days because of Discord's
// 5 rows limit), labelled in the member's timezone. With followup, both are follow-ups.
func sendDisposMenus(s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, cfg *guildconfig.GuildConfig, userID string, loc *time.Location, followup bool) error {
	m, layout := cfg.Messages(), cfg.SlotLayout
	week := svc.Week(cfg.Schedule)
	userSlots, err := svc.GetUserAvailability(context.Background(), i.GuildID, week, userID)
	if err != nil {
		log.Printf("error getting user availability: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	label := func(day, slot int) string { return svc.SlotLabelIn(m, layout, week, day, slot, loc) }

	first := &discordgo.WebhookParams{
		Content:    withLegend(m.DisposIntro, timezoneLegend(m, layout, loc)),
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: buildUserDisposSelectMenus(m, layout, userSlots, 0, 5, label),
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
	second := buildUserDisposSelectMenus(m, layout, userSlots, 5, 2, label)
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
	return ViewDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*ViewDisposHandler)(nil)

func (h *ViewDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	ctx := context.Background()
	week := h.svc.Week(cfg.Schedule)

	users, err := h.svc.GetAvailabilityUsers(ctx, i.GuildID, week)
	if err != nil {
		log.Printf("error getting availability users from button: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	unavailableUsers, err := h.svc.GetWeekUnavailableUsers(ctx, i.GuildID, week)
	if err != nil {
		log.Printf("error getting week unavailable users from button: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	description := h.svc.FormatUsersBySlot(m, cfg.SlotLayout, week, users)
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
				Color:       weeklyAccentColor,
			}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}
	for _, chunk := range chunks[1:] {
		if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{Description: chunk, Color: weeklyAccentColor}},
			Flags:  discordgo.MessageFlagsEphemeral,
		}); err != nil {
			return err
		}
	}
	return nil
}
