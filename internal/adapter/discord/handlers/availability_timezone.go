package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/gridimage"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// What to show once a member picked their timezone (in the picker custom ID).
const (
	tzNextDispos = "dispos" // the "Mes dispos" menus
	tzNextGrid   = "grid"   // the grid in their timezone
)

// Steps of the picker (last part of its custom ID): the UTC offset, west or east of UTC,
// then the country, whose timezone follows daylight saving time.
const (
	tzStepWest    = "w"
	tzStepEast    = "e"
	tzStepCountry = "c"
)

// utcOffsets are the offsets offered by the picker, in minutes, west to east
// (whole hours from UTC-12 to UTC+14, plus the half-hour ones in use).
var utcOffsets = []int{
	-12 * 60, -11 * 60, -10 * 60, -9*60 - 30, -9 * 60, -8 * 60, -7 * 60, -6 * 60, -5 * 60, -4 * 60,
	-3*60 - 30, -3 * 60, -2 * 60, -1 * 60,
	0, 1 * 60, 2 * 60, 3 * 60, 3*60 + 30, 4 * 60, 4*60 + 30, 5 * 60, 5*60 + 30, 5*60 + 45, 6 * 60,
	6*60 + 30, 7 * 60, 8 * 60, 9 * 60, 9*60 + 30, 10 * 60, 10*60 + 30, 11 * 60, 12 * 60, 13 * 60, 14 * 60,
}

// buildTimezonePicker asks for a UTC offset with two selects (Discord allows 25 options per
// select): west of UTC, then UTC and east. Each option shows the current time at that
// offset, to pick the one matching the clock. The selects' custom IDs are idBase:<step>.
func buildTimezonePicker(m *i18n.Messages, idBase string) []discordgo.MessageComponent {
	now := time.Now().UTC()
	countries := availability.CountriesByOffset(now)
	var west, east []discordgo.SelectMenuOption
	for _, minutes := range utcOffsets {
		name := availability.FormatUTCOffset(minutes * 60)
		opt := discordgo.SelectMenuOption{
			Label:       name + " · " + now.Add(time.Duration(minutes)*time.Minute).Format("15:04"),
			Value:       name,
			Description: countriesDescription(m, countries[minutes*60]),
		}
		if minutes < 0 {
			west = append(west, opt)
		} else {
			east = append(east, opt)
		}
	}
	row := func(side, placeholder string, options []discordgo.SelectMenuOption) discordgo.ActionsRow {
		return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
			CustomID:    idBase + ":" + side,
			Placeholder: placeholder,
			Options:     options,
		}}}
	}
	return []discordgo.MessageComponent{
		row(tzStepWest, m.PickerWest, west),
		row(tzStepEast, m.PickerEast, east),
	}
}

// buildCountryPicker asks for the country among the places at the UTC offset picked:
// storing the country's timezone lets daylight saving time apply by itself. The last
// option keeps the fixed offset, for places missing from the list.
func buildCountryPicker(m *i18n.Messages, idBase, offset string, places []availability.Timezone) []discordgo.MessageComponent {
	options := make([]discordgo.SelectMenuOption, 0, len(places)+1)
	seen := make(map[string]bool, len(places))
	for _, place := range places {
		if seen[place.Zone] || len(options) == 24 {
			continue
		}
		seen[place.Zone] = true
		options = append(options, discordgo.SelectMenuOption{Label: m.CountryName(place), Value: place.Zone})
	}
	options = append(options, discordgo.SelectMenuOption{
		Label:       fmt.Sprintf(m.OtherCountry, offset),
		Value:       offset,
		Description: m.OtherCountryHint,
	})
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
		CustomID:    idBase + ":" + tzStepCountry,
		Placeholder: m.CountryPlaceholder,
		Options:     options,
	}}}}
}

// countriesDescription lists places for a picker option, within Discord's 100 characters.
func countriesDescription(m *i18n.Messages, places []availability.Timezone) string {
	const limit = 100
	desc := ""
	for n, place := range places {
		name := m.CountryName(place)
		next := name
		if desc != "" {
			next = desc + ", " + name
		}
		if len([]rune(next)) > limit || (n < len(places)-1 && len([]rune(next)) > limit-2) {
			return desc + "…"
		}
		desc = next
	}
	return desc
}

// respondTimezonePicker answers an interaction with the timezone picker.
func respondTimezonePicker(s *discordgo.Session, i *discordgo.InteractionCreate, m *i18n.Messages, next string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    m.PickerContent,
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: buildTimezonePicker(m, TimezoneCustomIDPrefix+":"+next),
		},
	})
}

// editTimezoneButton reopens the timezone picker; next is what to show afterwards.
func editTimezoneButton(m *i18n.Messages, next string) discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{
		Label:    m.BtnEditTimezone,
		Style:    discordgo.SecondaryButton,
		CustomID: TimezoneEditCustomIDPrefix + ":" + next,
	}}}
}

// timezoneLegend reminds which timezone the menus use, for hourly layouts.
func timezoneLegend(m *i18n.Messages, layout availability.Layout, loc *time.Location) string {
	if !layout.IsHourly() || loc == nil {
		return ""
	}
	return fmt.Sprintf(m.TimezoneInMenus, m.TimezoneName(loc))
}

// sendLocalGrid sends the week's grid converted to the member's timezone, as an image or a
// text table depending on the guild's display.
func sendLocalGrid(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, svc *appavailability.Service, cfg *guildconfig.GuildConfig, loc *time.Location, followup bool) error {
	m := cfg.Messages()
	week := svc.Week(cfg.Schedule, cfg.Location)
	counts, err := svc.AvailabilitySummary(ctx, cfg.GuildID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability summary", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	view := svc.BuildWeeklyViewIn(m, cfg.SlotLayout, cfg.Location, cfg.Schedule, week, counts, loc)
	title := fmt.Sprintf(m.GridInYourTime, view.Title)

	params := &discordgo.WebhookParams{Flags: discordgo.MessageFlagsEphemeral}
	if cfg.Display == guildconfig.DisplayImage {
		img, err := gridimage.Render(view)
		if err != nil {
			slog.ErrorContext(ctx, "render local grid", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		accent := weekly.AccentColor
		params.Flags |= discordgo.MessageFlagsIsComponentsV2
		params.Files = []*discordgo.File{{Name: weekly.ImageName, ContentType: "image/png", Reader: bytes.NewReader(img)}}
		params.Components = []discordgo.MessageComponent{discordgo.Container{
			AccentColor: &accent,
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "### " + title + "\n-# " + view.Legend},
				discordgo.MediaGallery{Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://" + weekly.ImageName}}}},
				editTimezoneButton(m, tzNextGrid),
			},
		}}
	} else {
		params.Embeds = []*discordgo.MessageEmbed{{
			Title:       title,
			Description: "*" + view.Legend + "*\n" + view.Table,
			Color:       weekly.AccentColor,
		}}
		params.Components = []discordgo.MessageComponent{editTimezoneButton(m, tzNextGrid)}
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

// TimezoneSelectHandler handles the timezone picker: after the UTC offset, it asks for the
// country at that offset, stores its timezone, then shows what the member was opening
// (their dispos menus or the grid in their timezone).
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

func (h *TimezoneSelectHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	userID := getUserID(i)
	if userID == "" {
		return respondWithError(s, i, m.ErrUnknownUser)
	}
	// Custom ID: availability:tz:<next>:<step>
	next, step, _ := strings.Cut(strings.TrimPrefix(i.MessageComponentData().CustomID, TimezoneCustomIDPrefix+":"), ":")
	loc, err := readTimezonePick(ctx, s, i, m, TimezoneCustomIDPrefix+":"+next, step, m.PickerCountry, false)
	if loc == nil {
		return err
	}
	if err := h.svc.SetUserTimezone(ctx, userID, loc); err != nil {
		slog.ErrorContext(ctx, "set user timezone", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	if err := updateEphemeral(s, i, fmt.Sprintf(m.TimezoneSaved, m.TimezoneName(loc), m.Clock(time.Now().In(loc)))); err != nil {
		return err
	}

	switch next {
	case tzNextDispos:
		return sendDisposMenus(ctx, s, i, h.svc, cfg, userID, loc, true)
	case tzNextGrid:
		return sendLocalGrid(ctx, s, i, h.svc, cfg, loc, true)
	}
	return nil
}

// readTimezonePick reads a selection of the picker built on idBase, step being the last
// part of its custom ID. When a UTC offset was picked and countries have it, it asks for
// the country instead (countryPrompt, with the offset) and returns a nil location. v2 is
// whether the picker's message uses Components V2 (the /help panel).
func readTimezonePick(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, m *i18n.Messages, idBase, step, countryPrompt string, v2 bool) (*time.Location, error) {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return nil, respondWithError(s, i, m.ErrInvalid)
	}
	loc, err := availability.ParseTimezone(values[0])
	if err != nil {
		slog.WarnContext(ctx, "invalid timezone from picker", "value", values[0], "err", err)
		return nil, respondWithError(s, i, m.ErrInvalid)
	}
	if step == tzStepCountry {
		return loc, nil
	}
	_, offset := time.Now().In(loc).Zone()
	places := availability.CountriesByOffset(time.Now())[offset]
	if len(places) == 0 {
		return loc, nil
	}
	return nil, s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: pickerStep(v2, fmt.Sprintf(countryPrompt, values[0]), buildCountryPicker(m, idBase, values[0], places)),
	})
}

// pickerStep is a step of the timezone picker: a prompt above select menus. A Components
// V2 message has no content, so the prompt becomes a text component there.
func pickerStep(v2 bool, prompt string, rows []discordgo.MessageComponent) *discordgo.InteractionResponseData {
	if !v2 {
		return &discordgo.InteractionResponseData{Content: prompt, Components: rows}
	}
	return &discordgo.InteractionResponseData{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: append([]discordgo.MessageComponent{discordgo.TextDisplay{Content: prompt}}, rows...),
	}
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
	return weekly.MyTimeCustomID
}

var _ discord.InteractionHandler = (*MyTimeHandler)(nil)

func (h *MyTimeHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	userID := getUserID(i)
	if userID == "" {
		return respondWithError(s, i, m.ErrUnknownUser)
	}
	// Messages posted before the guild turned member timezones off still show the button.
	if !cfg.SlotLayout.IsHourly() {
		return respondWithError(s, i, m.NoHoursToConvert)
	}
	if !cfg.MemberTime() {
		return respondWithError(s, i, m.MemberTimeOff)
	}
	loc, err := h.svc.UserTimezone(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get user timezone", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	if loc == nil {
		return respondTimezonePicker(s, i, m, tzNextGrid)
	}
	return sendLocalGrid(ctx, s, i, h.svc, cfg, loc, false)
}

// TimezoneEditHandler handles "⚙️ Modifier mon fuseau": it shows the picker again.
type TimezoneEditHandler struct {
	guildCfg *appguildconfig.Service
}

// NewTimezoneEditHandler creates the "Modifier mon fuseau" handler.
func NewTimezoneEditHandler(guildCfg *appguildconfig.Service) *TimezoneEditHandler {
	return &TimezoneEditHandler{guildCfg: guildCfg}
}

func (h *TimezoneEditHandler) CustomID() string {
	return TimezoneEditCustomIDPrefix
}

var _ discord.InteractionHandler = (*TimezoneEditHandler)(nil)

func (h *TimezoneEditHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.guildCfg)
	if cfg == nil {
		return nil
	}
	next := strings.TrimPrefix(i.MessageComponentData().CustomID, TimezoneEditCustomIDPrefix+":")
	return respondTimezonePicker(s, i, cfg.Messages(), next)
}
