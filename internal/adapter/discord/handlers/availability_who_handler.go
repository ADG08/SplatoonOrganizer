package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// WhoDisposHandler handles the "Voir qui est dispo…" select of the weekly message.
type WhoDisposHandler struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
	week     *ViewDisposHandler
}

// NewWhoDisposHandler creates a new "who is available" handler.
func NewWhoDisposHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *WhoDisposHandler {
	return &WhoDisposHandler{svc: svc, guildCfg: guildCfg, week: NewViewDisposHandler(svc, guildCfg)}
}

func (h *WhoDisposHandler) CustomID() string {
	return weekly.WhoDisposCustomID
}

var _ discord.InteractionHandler = (*WhoDisposHandler)(nil)

func (h *WhoDisposHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	values := i.MessageComponentData().Values
	if len(values) > 0 && values[0] == weekly.WhoAllValue {
		return h.week.Handle(ctx, s, i)
	}
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	if len(values) == 0 {
		return respondWithError(s, i, m.ErrInvalid)
	}
	day, err := strconv.Atoi(values[0])
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return respondWithError(s, i, m.ErrInvalid)
	}

	week := h.svc.Week(cfg.Schedule)
	users, err := h.svc.AvailabilityUsers(ctx, i.GuildID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability users from select", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
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
						discordgo.TextDisplay{Content: "### " + fmt.Sprintf(m.WhoTitle, m.Days[day])},
						discordgo.TextDisplay{Content: h.svc.FormatDayUsers(m, cfg.SlotLayout, week, users, day)},
					},
				},
			},
		},
	})
}
