package handlers

import (
	"context"
	"log/slog"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/bwmarrin/discordgo"
)

// UnavailableDisposHandler handles the "Pas dispo cette semaine" button click.
type UnavailableDisposHandler struct {
	svc       *appavailability.Service
	teams     *appteam.Service
	refresher *weekly.Refresher
}

// NewUnavailableDisposHandler creates a new unavailable dispos handler.
func NewUnavailableDisposHandler(svc *appavailability.Service, teams *appteam.Service, refresher *weekly.Refresher) *UnavailableDisposHandler {
	return &UnavailableDisposHandler{svc: svc, teams: teams, refresher: refresher}
}

func (h *UnavailableDisposHandler) CustomID() string {
	return weekly.UnavailableCustomID
}

var _ discord.InteractionHandler = (*UnavailableDisposHandler)(nil)

func (h *UnavailableDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	t, _ := loadTeam(ctx, s, i, h.teams, weekly.UnavailableCustomID)
	if t == nil {
		return nil
	}
	m := t.Messages()
	userID := getUserID(i)
	if userID == "" {
		slog.WarnContext(ctx, "could not determine user ID", "guild", i.GuildID)
		return discord.RespondError(s, i, m.ErrUnknownUser)
	}

	nowUnavailable, err := h.svc.ToggleWeekUnavailable(ctx, t.ID, h.svc.Week(t.Schedule, t.Location), userID)
	if err != nil {
		slog.ErrorContext(ctx, "toggle week unavailability", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	content := withTeam(t, m.AvailableAgain)
	if nowUnavailable {
		content = withTeam(t, m.UnavailableNow)
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
	h.refresher.Request(s, t.ID)

	return nil
}
