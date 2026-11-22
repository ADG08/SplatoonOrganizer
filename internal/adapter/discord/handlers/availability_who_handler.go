package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// WhoDisposHandler handles the "Voir qui est dispo…" select of the weekly message.
type WhoDisposHandler struct {
	svc   *appavailability.Service
	teams *appteam.Service
	week  *ViewDisposHandler
}

// NewWhoDisposHandler creates a new "who is available" handler.
func NewWhoDisposHandler(svc *appavailability.Service, teams *appteam.Service) *WhoDisposHandler {
	return &WhoDisposHandler{svc: svc, teams: teams, week: NewViewDisposHandler(svc, teams)}
}

func (h *WhoDisposHandler) CustomID() string {
	return weekly.WhoDisposCustomID
}

var _ discord.InteractionHandler = (*WhoDisposHandler)(nil)

func (h *WhoDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	t, _ := loadTeam(ctx, s, i, h.teams, weekly.WhoDisposCustomID)
	if t == nil {
		return nil
	}
	values := i.MessageComponentData().Values
	if len(values) > 0 && values[0] == weekly.WhoAllValue {
		return h.week.show(ctx, s, i, t)
	}
	m := t.Messages()
	if len(values) == 0 {
		return discord.RespondError(s, i, m.ErrInvalid)
	}
	day, err := strconv.Atoi(values[0])
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return discord.RespondError(s, i, m.ErrInvalid)
	}

	week := h.svc.Week(t.Schedule, t.Location)
	users, err := h.svc.AvailabilityUsers(ctx, t.ID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability users from select", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	accent := weekly.AccentColor
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:           discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Components: []discordgo.MessageComponent{
				discordgo.Container{
					AccentColor: &accent,
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{Content: "### " + weekly.Title(t, fmt.Sprintf(m.WhoTitle, m.Days[day]))},
						discordgo.TextDisplay{Content: h.svc.FormatDayUsers(m, t.SlotLayout, t.Location, week, users, day)},
					},
				},
			},
		},
	})
}
