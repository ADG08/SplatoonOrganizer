package handlers

import (
	"context"
	"log/slog"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// UnavailableDisposHandler handles the "Pas dispo cette semaine" button click.
type UnavailableDisposHandler struct {
	svc       *appavailability.Service
	guildCfg  *appguildconfig.Service
	refresher *weekly.Refresher
}

// NewUnavailableDisposHandler creates a new unavailable dispos handler.
func NewUnavailableDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service, refresher *weekly.Refresher) *UnavailableDisposHandler {
	return &UnavailableDisposHandler{svc: svc, guildCfg: guildCfg, refresher: refresher}
}

func (h *UnavailableDisposHandler) CustomID() string {
	return weekly.UnavailableCustomID
}

var _ discord.InteractionHandler = (*UnavailableDisposHandler)(nil)

func (h *UnavailableDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	nowUnavailable, err := h.svc.ToggleWeekUnavailable(ctx, i.GuildID, h.svc.Week(cfg.Schedule), userID)
	if err != nil {
		slog.ErrorContext(ctx, "toggle week unavailability", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	content := m.AvailableAgain
	if nowUnavailable {
		content = m.UnavailableNow
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		slog.ErrorContext(ctx, "respond to unavailable toggle", "guild", i.GuildID, "err", err)
	}

	// Marking unavailable clears slots, so refresh the main weekly table (best-effort).
	h.refresher.Request(s, i.GuildID)

	return nil
}
