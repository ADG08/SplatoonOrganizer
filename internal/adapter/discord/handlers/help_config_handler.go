package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const HelpConfigCustomIDPrefix = "help:config"

// HelpConfigHandler handles the buttons, forms and timezone picker of the /help panel.
type HelpConfigHandler struct {
	svc        *appguildconfig.Service
	availSvc   *appavailability.Service
	refresher  *weekly.Refresher // updates the weekly message when a change doesn't need a new one
	ownerID    string
	supportURL string // donation page linked from the panel, optional
}

// NewHelpConfigHandler creates a new help config handler. ownerID is optional.
func NewHelpConfigHandler(svc *appguildconfig.Service, availSvc *appavailability.Service, refresher *weekly.Refresher, ownerID, supportURL string) *HelpConfigHandler {
	return &HelpConfigHandler{svc: svc, availSvc: availSvc, refresher: refresher, ownerID: ownerID, supportURL: supportURL}
}

func (h *HelpConfigHandler) CustomID() string {
	return HelpConfigCustomIDPrefix
}

var _ discord.InteractionHandler = (*HelpConfigHandler)(nil)

func (h *HelpConfigHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(ctx, s, i, h.svc)
	if cfg == nil {
		return nil
	}
	m := cfg.Messages()
	if !isAdminOrOwnerInteraction(i, h.ownerID) {
		return respondWithError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return respondWithError(s, i, m.ErrGuildOnly)
	}

	if i.Type == discordgo.InteractionModalSubmit {
		data := i.ModalSubmitData()
		switch data.CustomID {
		case discord.HelpPanelModalPublishID:
			return h.handlePublishSubmit(ctx, s, i, cfg, data)
		case discord.HelpPanelModalSlotsID:
			return h.handleSlotsSubmit(ctx, s, i, cfg, data)
		case discord.HelpPanelModalMessageID:
			return h.handleMessageSubmit(ctx, s, i, cfg, data)
		}
		return respondWithError(s, i, m.ErrInvalid)
	}

	customID := i.MessageComponentData().CustomID
	if step, ok := strings.CutPrefix(customID, discord.HelpPanelTimezoneID+":"); ok {
		return h.handleTimezonePick(ctx, s, i, cfg, step)
	}
	modal := func(customID, title string, fields []discordgo.MessageComponent) error {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{CustomID: customID, Title: title, Components: fields},
		})
	}
	switch customID {
	case discord.HelpPanelOpenPublishID:
		return modal(discord.HelpPanelModalPublishID, m.SectionPublish, buildPublishModalComponents(cfg))
	case discord.HelpPanelOpenSlotsID:
		return modal(discord.HelpPanelModalSlotsID, m.SectionSlots, buildSlotsModalComponents(cfg))
	case discord.HelpPanelOpenMessageID:
		return modal(discord.HelpPanelModalMessageID, m.SectionMessage, buildMessageModalComponents(cfg))
	case discord.HelpPanelOpenTimezoneID:
		// The panel turns into the picker, and back into the panel once a timezone is picked.
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: pickerStep(true, fmt.Sprintf(m.GuildPickerContent, m.TimezoneName(cfg.Location)), buildTimezonePicker(m, discord.HelpPanelTimezoneID)),
		})
	}
	return respondWithError(s, i, m.ErrInvalid)
}

// buildPublishModalComponents builds the publication form, prefilled with the current
// channel, role and schedule. The role is optional: unselecting it stops the mentions.
func buildPublishModalComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	days := []discordgo.SelectMenuOption{
		{Label: m.PublishSunday, Value: strconv.Itoa(int(time.Sunday)), Default: cfg.Schedule.Weekday == time.Sunday},
		{Label: m.PublishMonday, Value: strconv.Itoa(int(time.Monday)), Default: cfg.Schedule.Weekday == time.Monday},
	}
	hours := make([]discordgo.SelectMenuOption, 24)
	for h := range hours {
		hours[h] = discordgo.SelectMenuOption{Label: m.HourLabel(h), Value: strconv.Itoa(h), Default: h == cfg.Schedule.Hour}
	}
	return []discordgo.MessageComponent{
		discordgo.Label{
			Label:       m.ChannelLabel,
			Description: m.ChannelHint,
			Component: discordgo.SelectMenu{
				CustomID:      "channel",
				MenuType:      discordgo.ChannelSelectMenu,
				ChannelTypes:  []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				MinValues:     new(1),
				MaxValues:     1,
				Required:      new(true),
				DefaultValues: defaultChannelValues(cfg.ChannelID),
			},
		},
		discordgo.Label{
			Label:       m.RoleLabel,
			Description: m.RoleHint,
			Component: discordgo.SelectMenu{
				CustomID:      "role",
				MenuType:      discordgo.RoleSelectMenu,
				MinValues:     new(0),
				MaxValues:     1,
				Required:      new(false),
				DefaultValues: defaultRoleValues(cfg.RoleID),
			},
		},
		selectField(m.PublishDayLabel, m.PublishDayHint, "publish_weekday", true, days),
		selectField(m.PublishHourLabel, m.PublishHourHint, "publish_hour", true, hours),
	}
}

// Values of the slots form's member time select.
const (
	memberTimeServer = "server" // everyone sees the guild's time
	memberTimeOwn    = "own"    // each member sees their own timezone
)

// buildSlotsModalComponents builds the slots form, prefilled with the current layout.
func buildSlotsModalComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	current := cfg.SlotLayout
	start, end := availability.DefaultStartHour, availability.DefaultEndHour
	if current.IsHourly() {
		start, end = current.StartHour, current.EndHour
	}
	modes := []discordgo.SelectMenuOption{
		{Label: m.LayoutSummary(availability.HalfDayLayout()), Value: string(availability.ModeHalfDay), Default: !current.IsHourly()},
		{Label: m.ModeHourly, Value: string(availability.ModeHourly), Default: current.IsHourly()},
	}
	// Discord allows 25 options per select menu: 0h..23h for the start, and for the end
	// 5h..minuit then 1h..5h the next day (an evening going on after midnight).
	var startOptions, endOptions []discordgo.SelectMenuOption
	for h := range 24 {
		startOptions = append(startOptions, discordgo.SelectMenuOption{
			Label: m.HourLabel(h), Value: strconv.Itoa(h), Default: h == start,
		})
	}
	for h := availability.MaxEndHour - 24; h <= availability.MaxEndHour; h++ {
		endOptions = append(endOptions, discordgo.SelectMenuOption{
			Label: m.UntilLabel(h), Value: strconv.Itoa(h), Default: h == end,
		})
	}
	memberTime := []discordgo.SelectMenuOption{
		{Label: m.MemberTimeServer, Value: memberTimeServer, Default: !cfg.MemberTimezones},
		{Label: m.MemberTimeOwn, Value: memberTimeOwn, Default: cfg.MemberTimezones},
	}
	return []discordgo.MessageComponent{
		selectField(m.ModeLabel, m.ModeHint, "slot_mode", true, modes),
		// The hours are optional: they don't apply to "Après-midi / Soir".
		selectField(m.StartHourLabel, m.StartHourHint, "slot_start_hour", false, startOptions),
		selectField(m.EndHourLabel, m.EndHourHint, "slot_end_hour", false, endOptions),
		selectField(m.MemberTimeLabel, m.MemberTimeHint, "member_time", true, memberTime),
	}
}

// buildMessageModalComponents builds the message form: display, language, and the
// players from which a slot is green.
func buildMessageModalComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	languages := make([]discordgo.SelectMenuOption, len(i18n.Langs))
	for n, lang := range i18n.Langs {
		languages[n] = discordgo.SelectMenuOption{Label: m.LanguageNames[n], Value: string(lang), Default: lang == i18n.Parse(string(cfg.Language))}
	}
	displays := []discordgo.SelectMenuOption{
		{Label: m.DisplayClassic, Value: string(guildconfig.DisplayClassic), Default: cfg.Display != guildconfig.DisplayImage},
		{Label: m.DisplayImage, Value: string(guildconfig.DisplayImage), Default: cfg.Display == guildconfig.DisplayImage},
	}
	fullAt := []discordgo.SelectMenuOption{{Label: m.FullAtAuto, Value: "0", Default: cfg.FullAt <= 0}}
	for n := 1; n <= availability.MaxFullAt; n++ {
		fullAt = append(fullAt, discordgo.SelectMenuOption{Label: m.Players(n), Value: strconv.Itoa(n), Default: cfg.FullAt == n})
	}
	return []discordgo.MessageComponent{
		selectField(m.DisplayLabel, m.DisplayHint, "display_style", true, displays),
		selectField(m.LanguageLabel, "", "language", true, languages),
		selectField(m.FullAtLabel, m.FullAtHint, "full_at", true, fullAt),
	}
}

// selectField is a modal field holding a single-choice select menu.
func selectField(label, hint, customID string, required bool, options []discordgo.SelectMenuOption) discordgo.Label {
	minValues := 0
	if required {
		minValues = 1
	}
	return discordgo.Label{
		Label:       label,
		Description: hint,
		Component: discordgo.SelectMenu{
			CustomID:  customID,
			MinValues: new(minValues),
			MaxValues: 1,
			Required:  new(required),
			Options:   options,
		},
	}
}

// handlePublishSubmit saves the channel, role and schedule. A new channel (or the same
// one picked again for a paused guild) gets the weekly message right away; a new schedule
// does when its publication time is already past.
func (h *HelpConfigHandler) handlePublishSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *guildconfig.GuildConfig, data discordgo.ModalSubmitInteractionData) error {
	m := previous.Messages()
	channelID := modalSelectValue(data.Components, "channel")
	roleID := modalSelectValue(data.Components, "role")
	weekday, errDay := strconv.Atoi(modalSelectValue(data.Components, "publish_weekday"))
	hour, errHour := strconv.Atoi(modalSelectValue(data.Components, "publish_hour"))
	schedule, err := availability.NewSchedule(time.Weekday(weekday), hour)
	if channelID == "" || errDay != nil || errHour != nil || err != nil {
		return respondWithError(s, i, m.ErrInvalid)
	}

	channelChanged := channelID != previous.ChannelID || !previous.Active
	if !channelChanged && roleID == previous.RoleID && schedule == previous.Schedule {
		return h.refreshPanel(ctx, s, i, m.NothingChanged)
	}
	if channelChanged {
		if err := h.svc.SetChannel(ctx, previous.GuildID, channelID); err != nil {
			slog.ErrorContext(ctx, "help config: set channel", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	}
	if roleID != previous.RoleID {
		if err := h.svc.SetRole(ctx, previous.GuildID, roleID); err != nil {
			slog.ErrorContext(ctx, "help config: set role", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	}
	if schedule != previous.Schedule {
		if err := h.svc.SetSchedule(ctx, previous.GuildID, schedule); err != nil {
			slog.ErrorContext(ctx, "help config: set schedule", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	}

	cfg := *previous
	cfg.ChannelID, cfg.RoleID, cfg.Schedule, cfg.Active = channelID, roleID, schedule, true
	republish := channelChanged
	if !republish {
		if republish, err = weekly.Due(ctx, h.availSvc, &cfg); err != nil {
			slog.ErrorContext(ctx, "help config: check weekly message", "guild", i.GuildID, "err", err)
		}
	}
	return h.saved(ctx, s, i, &cfg, m.ConfigSaved, republish)
}

// handleSlotsSubmit saves the slot layout and whose time the members see. A new layout
// starts the week over, as slot indexes of the old one would be misread with it.
func (h *HelpConfigHandler) handleSlotsSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *guildconfig.GuildConfig, data discordgo.ModalSubmitInteractionData) error {
	m := previous.Messages()
	layout := availability.HalfDayLayout()
	if modalSelectValue(data.Components, "slot_mode") == string(availability.ModeHourly) {
		// An empty hour falls back to the default 20h - minuit.
		start, end := availability.DefaultStartHour, availability.DefaultEndHour
		if v := modalSelectValue(data.Components, "slot_start_hour"); v != "" {
			start, _ = strconv.Atoi(v)
		}
		if v := modalSelectValue(data.Components, "slot_end_hour"); v != "" {
			end, _ = strconv.Atoi(v)
		}
		var err error
		if layout, err = availability.HourlyLayout(start, end); err != nil {
			return respondWithError(s, i, m.HourOrder)
		}
	}
	memberTimezones := modalSelectValue(data.Components, "member_time") == memberTimeOwn
	layoutChanged := layout != previous.SlotLayout
	if !layoutChanged && memberTimezones == previous.MemberTimezones {
		return h.refreshPanel(ctx, s, i, m.NothingChanged)
	}

	message := m.ConfigSaved
	if layoutChanged {
		if err := h.svc.SetMessageSettings(ctx, previous.GuildID, layout, previous.Display, previous.Language); err != nil {
			slog.ErrorContext(ctx, "help config: set slots", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		if err := h.availSvc.ResetWeek(ctx, previous.GuildID, h.availSvc.Week(previous.Schedule, previous.Location)); err != nil {
			slog.ErrorContext(ctx, "help config: reset week", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		message += m.SlotsReset
	}
	if memberTimezones != previous.MemberTimezones {
		if err := h.svc.SetMemberTimezones(ctx, previous.GuildID, memberTimezones); err != nil {
			slog.ErrorContext(ctx, "help config: set member timezones", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	}

	cfg := *previous
	cfg.SlotLayout, cfg.MemberTimezones = layout, memberTimezones
	// The message changes with the layout, or with the "Mon fuseau" button of hourly slots.
	republish := layoutChanged || cfg.MemberTime() != previous.MemberTime()
	return h.saved(ctx, s, i, &cfg, message, republish)
}

// handleMessageSubmit saves the display, the language and the green threshold. A new
// display or language posts the message again (it may switch between an embed and
// Components V2, which an edit can't do); a new threshold only redraws it.
func (h *HelpConfigHandler) handleMessageSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *guildconfig.GuildConfig, data discordgo.ModalSubmitInteractionData) error {
	lang := i18n.Parse(modalSelectValue(data.Components, "language"))
	m := i18n.For(lang) // answer in the language just picked
	display := guildconfig.DisplayClassic
	if modalSelectValue(data.Components, "display_style") == string(guildconfig.DisplayImage) {
		display = guildconfig.DisplayImage
	}
	fullAt, err := strconv.Atoi(modalSelectValue(data.Components, "full_at"))
	if err != nil || fullAt < 0 || fullAt > availability.MaxFullAt {
		return respondWithError(s, i, m.ErrInvalid)
	}
	republish := display != previous.Display || lang != previous.Language
	if !republish && fullAt == previous.FullAt {
		return h.refreshPanel(ctx, s, i, m.NothingChanged)
	}

	if republish {
		if err := h.svc.SetMessageSettings(ctx, previous.GuildID, previous.SlotLayout, display, lang); err != nil {
			slog.ErrorContext(ctx, "help config: set message settings", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	}
	if fullAt != previous.FullAt {
		if err := h.svc.SetFullAt(ctx, previous.GuildID, fullAt); err != nil {
			slog.ErrorContext(ctx, "help config: set full at", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		if !republish {
			// Only the colors change: redraw the message, without a new one pinging the role.
			h.refresher.Request(s, previous.GuildID)
		}
	}

	cfg := *previous
	cfg.Display, cfg.Language, cfg.FullAt = display, lang, fullAt
	return h.saved(ctx, s, i, &cfg, m.ConfigSaved, republish)
}

// handleTimezonePick saves the guild's timezone picked in the picker. Hourly slots move
// with it, so the week's availabilities are reset as for a new slot layout.
func (h *HelpConfigHandler) handleTimezonePick(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *guildconfig.GuildConfig, step string) error {
	m := previous.Messages()
	loc, err := readTimezonePick(ctx, s, i, m, discord.HelpPanelTimezoneID, step, m.GuildPickerCountry, true)
	if loc == nil {
		return err
	}
	if loc.String() == previous.Location.String() {
		return h.refreshPanel(ctx, s, i, m.NothingChanged)
	}
	if err := h.svc.SetTimezone(ctx, previous.GuildID, loc); err != nil {
		slog.ErrorContext(ctx, "help config: set timezone", "guild", i.GuildID, "err", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	message := fmt.Sprintf(m.GuildTimezoneSaved, m.TimezoneName(loc), m.Clock(time.Now().In(loc)))
	republish := previous.SlotLayout.IsHourly()
	if republish {
		if err := h.availSvc.ResetWeek(ctx, previous.GuildID, h.availSvc.Week(previous.Schedule, previous.Location)); err != nil {
			slog.ErrorContext(ctx, "help config: reset week", "guild", i.GuildID, "err", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		message += m.SlotsReset
	}

	cfg := *previous
	cfg.Location = loc
	// Half-day slots don't move: republish only when the current week changed.
	if !republish && cfg.ChannelID != "" {
		if republish, err = weekly.Due(ctx, h.availSvc, &cfg); err != nil {
			slog.ErrorContext(ctx, "help config: check weekly message", "guild", i.GuildID, "err", err)
		}
	}
	return h.saved(ctx, s, i, &cfg, message, republish)
}

// saved shows the panel again after a change, once the weekly message is being posted
// again when asked (in the background: Discord wants the answer within 3 seconds).
func (h *HelpConfigHandler) saved(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, cfg *guildconfig.GuildConfig, message string, republish bool) error {
	if republish && cfg.ChannelID != "" {
		message += cfg.Messages().PublishedNow
		weekly.PublishInBackground(ctx, s, h.availSvc, cfg)
	}
	return h.refreshPanel(ctx, s, i, message)
}

// modalSelectValue returns the first selected value of a modal select menu, or "".
func modalSelectValue(components []discordgo.MessageComponent, customID string) string {
	menu := findModalSelectMenu(components, customID)
	if menu == nil || len(menu.Values) == 0 {
		return ""
	}
	return menu.Values[0]
}

// refreshPanel shows the panel again with the saved settings, in the guild's language.
func (h *HelpConfigHandler) refreshPanel(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, description string) error {
	cfg := loadGuildConfig(ctx, s, i, h.svc)
	if cfg == nil {
		return nil
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsIsComponentsV2,
			Components: discord.BuildHelpPanel(cfg, description, h.supportURL),
		},
	})
}

func defaultChannelValues(channelID string) []discordgo.SelectMenuDefaultValue {
	if channelID == "" {
		return nil
	}
	return []discordgo.SelectMenuDefaultValue{
		{
			ID:   channelID,
			Type: discordgo.SelectMenuDefaultValueChannel,
		},
	}
}

func defaultRoleValues(roleID string) []discordgo.SelectMenuDefaultValue {
	if roleID == "" {
		return nil
	}
	return []discordgo.SelectMenuDefaultValue{
		{
			ID:   roleID,
			Type: discordgo.SelectMenuDefaultValueRole,
		},
	}
}

func findModalSelectMenu(components []discordgo.MessageComponent, customID string) *discordgo.SelectMenu {
	for _, comp := range components {
		switch c := comp.(type) {
		case discordgo.SelectMenu:
			if c.CustomID == customID {
				menu := c
				return &menu
			}
		case *discordgo.SelectMenu:
			if c.CustomID == customID {
				return c
			}
		case discordgo.ActionsRow:
			if menu := findModalSelectMenu(c.Components, customID); menu != nil {
				return menu
			}
		case *discordgo.ActionsRow:
			if menu := findModalSelectMenu(c.Components, customID); menu != nil {
				return menu
			}
		case discordgo.Label:
			if menu := findModalSelectMenu([]discordgo.MessageComponent{c.Component}, customID); menu != nil {
				return menu
			}
		case *discordgo.Label:
			if menu := findModalSelectMenu([]discordgo.MessageComponent{c.Component}, customID); menu != nil {
				return menu
			}
		}
	}
	return nil
}

func isAdminOrOwnerInteraction(i *discordgo.InteractionCreate, ownerID string) bool {
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	if ownerID != "" && i.Member.User != nil && i.Member.User.ID == ownerID {
		return true
	}
	return false
}
