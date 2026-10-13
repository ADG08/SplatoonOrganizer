package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/gridimage"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// What to show once a member picked their timezone (in the picker custom ID).
const (
	tzNextDispos = "dispos" // the "Mes dispos" menus
	tzNextGrid   = "grid"   // the grid in their timezone
)

// utcOffsets are the offsets offered by the picker, in minutes, west to east
// (whole hours from UTC-12 to UTC+14, plus the half-hour ones in use).
var utcOffsets = []int{
	-12 * 60, -11 * 60, -10 * 60, -9*60 - 30, -9 * 60, -8 * 60, -7 * 60, -6 * 60, -5 * 60, -4 * 60,
	-3*60 - 30, -3 * 60, -2 * 60, -1 * 60,
	0, 1 * 60, 2 * 60, 3 * 60, 3*60 + 30, 4 * 60, 4*60 + 30, 5 * 60, 5*60 + 30, 5*60 + 45, 6 * 60,
	6*60 + 30, 7 * 60, 8 * 60, 9 * 60, 9*60 + 30, 10 * 60, 10*60 + 30, 11 * 60, 12 * 60, 13 * 60, 14 * 60,
}

// buildTimezonePicker asks for the member's UTC offset with two selects (Discord allows 25
// options per select): west of UTC, then UTC and east. Each option shows the current time
// at that offset, so members can pick the one matching their clock.
func buildTimezonePicker(next string) []discordgo.MessageComponent {
	now := time.Now().UTC()
	countries := availability.CountriesByOffset(now)
	var west, east []discordgo.SelectMenuOption
	for _, minutes := range utcOffsets {
		name := availability.FormatUTCOffset(minutes * 60)
		opt := discordgo.SelectMenuOption{
			Label:       name + " · " + now.Add(time.Duration(minutes)*time.Minute).Format("15:04"),
			Value:       name,
			Description: countriesDescription(countries[minutes*60]),
		}
		if minutes < 0 {
			west = append(west, opt)
		} else {
			east = append(east, opt)
		}
	}
	row := func(side, placeholder string, options []discordgo.SelectMenuOption) discordgo.ActionsRow {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
			CustomID:    TimezoneCustomIDPrefix + ":" + next + ":" + side,
			Placeholder: placeholder,
			Options:     options,
		}}}
	}
	return []discordgo.MessageComponent{
		row("w", "UTC-12 → UTC-1 · Amériques, Pacifique", west),
		row("e", "UTC+0 → UTC+14 · Europe, Afrique, Asie, Océanie", east),
	}
}

// countriesDescription lists places for a picker option, within Discord's 100 characters.
func countriesDescription(names []string) string {
	const limit = 100
	desc := ""
	for n, name := range names {
		next := name
		if desc != "" {
			next = desc + ", " + name
		}
		if len([]rune(next)) > limit || (n < len(names)-1 && len([]rune(next)) > limit-2) {
			return desc + "…"
		}
		desc = next
	}
	return desc
}

const timezonePickerContent = "🌍 **Quel est ton fuseau horaire ?**\n" +
	"Choisis ton décalage UTC (l'heure affichée doit être ton heure actuelle) : " +
	"les créneaux s'afficheront ensuite à ton heure.\n" +
	"-# Un décalage fixe ne suit pas l'heure d'été : pense à le changer quand l'heure change, " +
	"ou utilise `/fuseau` avec ta ville (ex : `Europe/Paris`) pour que ce soit automatique."

// respondTimezonePicker answers an interaction with the timezone picker.
func respondTimezonePicker(s *discordgo.Session, i *discordgo.InteractionCreate, next string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    timezonePickerContent,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: buildTimezonePicker(next),
		},
	})
}

// editTimezoneButton reopens the timezone picker; next is what to show afterwards.
func editTimezoneButton(next string) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{
		Label:    "⚙️ Modifier mon fuseau",
		Style:    discordgo.SecondaryButton,
		CustomID: TimezoneEditCustomIDPrefix + ":" + next,
	}}}
}

// timezoneLegend reminds which timezone the menus use, for hourly layouts.
func timezoneLegend(layout availability.Layout, loc *time.Location) string {
	if !layout.IsHourly() || loc == nil {
		return ""
	}
	return "-# 🌍 Heures affichées à ton heure (" + availability.TimezoneName(loc) + ")"
}

// sendLocalGrid sends the week's grid converted to the member's timezone, as an image or a
// text table depending on the guild's display.
func sendLocalGrid(s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, cfg *guildconfig.GuildConfig, loc *time.Location, followup bool) error {
	week := svc.CurrentWeek()
	counts, err := svc.GetAvailabilitySummary(context.Background(), cfg.GuildID, week)
	if err != nil {
		log.Printf("error getting availability summary: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération des disponibilités.")
	}
	view := svc.BuildWeeklyViewIn(cfg.SlotLayout, week, counts, loc)

	params := &discordgo.WebhookParams{Flags: discordgo.MessageFlagsEphemeral}
	if cfg.Display == guildconfig.DisplayImage {
		img, err := gridimage.Render(view)
		if err != nil {
			log.Printf("error rendering local grid: %v", err)
			return respondWithError(s, i, "Erreur lors de la génération de l'image.")
		}
		accent := weeklyAccentColor
		params.Flags |= discordgo.MessageFlagsIsComponentsV2
		params.Files = []*discordgo.File{{Name: weeklyImageName, ContentType: "image/webp", Reader: bytes.NewReader(img)}}
		params.Components = []discordgo.MessageComponent{discordgo.Container{
			AccentColor: &accent,
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "### " + view.Title + " · à ton heure\n-# " + view.Legend},
				discordgo.MediaGallery{Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://" + weeklyImageName}}}},
				editTimezoneButton(tzNextGrid),
			},
		}}
	} else {
		params.Embeds = []*discordgo.MessageEmbed{{
			Title:       view.Title + " · à ton heure",
			Description: "*" + view.Legend + "*\n" + view.Table,
			Color:       weeklyAccentColor,
		}}
		params.Components = []discordgo.MessageComponent{editTimezoneButton(tzNextGrid)}
	}

	if followup {
		_, err = s.FollowupMessageCreate(i.Interaction, true, params)
		return err
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: params.Flags, Embeds: params.Embeds, Components: params.Components, Files: params.Files,
		},
	})
}

// TimezoneSelectHandler stores the timezone picked in the picker, then shows what the
// member was opening (their dispos menus or the grid in their timezone).
type TimezoneSelectHandler struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewTimezoneSelectHandler creates the timezone picker handler.
func NewTimezoneSelectHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *TimezoneSelectHandler {
	return &TimezoneSelectHandler{svc: svc, guildCfg: guildCfg}
}

func (h *TimezoneSelectHandler) CustomID() string {
	return TimezoneCustomIDPrefix
}

var _ discord.InteractionHandler = (*TimezoneSelectHandler)(nil)

func (h *TimezoneSelectHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	values := i.MessageComponentData().Values
	if userID == "" || len(values) == 0 {
		return respondWithError(s, i, "Interaction invalide.")
	}
	// Custom ID: availability:tz:<next>:<w|e>
	next, _, _ := strings.Cut(strings.TrimPrefix(i.MessageComponentData().CustomID, TimezoneCustomIDPrefix+":"), ":")
	loc, err := availability.ParseTimezone(values[0])
	if err != nil {
		log.Printf("timezone picker: %q: %v", values[0], err)
		return respondWithError(s, i, "Fuseau inconnu.")
	}
	ctx := context.Background()
	if err := h.svc.SetUserTimezone(ctx, userID, loc); err != nil {
		log.Printf("error setting user timezone: %v", err)
		return respondWithError(s, i, "Erreur lors de l'enregistrement de ton fuseau.")
	}
	if err := updateEphemeral(s, i, fmt.Sprintf("✅ Fuseau enregistré : **%s** (il est %s chez toi).",
		availability.TimezoneName(loc), time.Now().In(loc).Format("15:04"))); err != nil {
		return err
	}

	cfg, err := h.guildCfg.Get(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting guild config: %v", err)
		return nil
	}
	switch next {
	case tzNextDispos:
		return sendDisposMenus(s, i, h.svc, cfg.SlotLayout, userID, loc, true)
	case tzNextGrid:
		return sendLocalGrid(s, i, h.svc, cfg, loc, true)
	}
	return nil
}

// updateEphemeral replaces the content of the ephemeral message holding the component.
func updateEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: content, Components: []discordgo.MessageComponent{}},
	})
}

// MyTimeHandler handles the "🌍 Mon fuseau" button: the week's grid in the member's timezone.
type MyTimeHandler struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewMyTimeHandler creates the "Mon fuseau" handler.
func NewMyTimeHandler(svc *appavailability.Service, guildCfg *appguildconfig.Service) *MyTimeHandler {
	return &MyTimeHandler{svc: svc, guildCfg: guildCfg}
}

func (h *MyTimeHandler) CustomID() string {
	return MyTimeCustomIDPrefix
}

var _ discord.InteractionHandler = (*MyTimeHandler)(nil)

func (h *MyTimeHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	ctx := context.Background()
	cfg, err := h.guildCfg.Get(ctx, i.GuildID)
	if err != nil {
		log.Printf("error getting guild config: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration du serveur.")
	}
	if !cfg.SlotLayout.IsHourly() {
		return respondWithError(s, i, "Les créneaux de ce serveur sont « Après-midi / Soir » : il n'y a pas d'heure à convertir.")
	}
	loc, err := h.svc.UserTimezone(ctx, userID)
	if err != nil {
		log.Printf("error getting user timezone: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de ton fuseau.")
	}
	if loc == nil {
		return respondTimezonePicker(s, i, tzNextGrid)
	}
	return sendLocalGrid(s, i, h.svc, cfg, loc, false)
}

// TimezoneEditHandler handles "⚙️ Modifier mon fuseau": it shows the picker again.
type TimezoneEditHandler struct{}

// NewTimezoneEditHandler creates the "Modifier mon fuseau" handler.
func NewTimezoneEditHandler() *TimezoneEditHandler {
	return &TimezoneEditHandler{}
}

func (h *TimezoneEditHandler) CustomID() string {
	return TimezoneEditCustomIDPrefix
}

var _ discord.InteractionHandler = (*TimezoneEditHandler)(nil)

func (h *TimezoneEditHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	next := strings.TrimPrefix(i.MessageComponentData().CustomID, TimezoneEditCustomIDPrefix+":")
	return respondTimezonePicker(s, i, next)
}
