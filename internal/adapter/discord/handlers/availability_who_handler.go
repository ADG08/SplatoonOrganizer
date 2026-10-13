package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
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
	return WhoDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*WhoDisposHandler)(nil)

func (h *WhoDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return respondWithError(s, i, "Interaction invalide.")
	}
	if values[0] == whoDisposAllValue {
		return h.week.Handle(s, i)
	}
	day, err := strconv.Atoi(values[0])
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return respondWithError(s, i, "Interaction invalide.")
	}

	ctx := context.Background()
	week := h.svc.CurrentWeek()
	users, err := h.svc.GetAvailabilityUsers(ctx, week)
	if err != nil {
		log.Printf("error getting availability users from select: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération des disponibilités.")
	}
	layout, err := h.guildCfg.SlotLayout(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting slot layout: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration du serveur.")
	}

	accent := weeklyAccentColor
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:           discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Components: []discordgo.MessageComponent{
				discordgo.Container{
					AccentColor: &accent,
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{Content: "### Qui est dispo " + appavailability.DayLabel(day) + " ?"},
						discordgo.TextDisplay{Content: h.svc.FormatDayUsers(layout, week, users, day)},
					},
				},
			},
		},
	})
}
