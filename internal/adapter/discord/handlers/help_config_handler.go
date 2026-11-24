package handlers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguild "github.com/ADG08/SplatoonOrganizer/internal/application/guild"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// HelpConfigHandler handles the buttons, forms and timezone picker of the /help panels.
type HelpConfigHandler struct {
	teams      *appteam.Service
	guilds     *appguild.Service // the language of what isn't a team's
	availSvc   *appavailability.Service
	refresher  *weekly.Refresher // updates the weekly message when a change doesn't need a new one
	welcome    *WelcomeHandler   // updates the welcome message as the setup goes on
	ownerID    string
	supportURL string // donation page linked from the panel, optional
}

// NewHelpConfigHandler creates a new help config handler. ownerID is optional.
func NewHelpConfigHandler(teams *appteam.Service, guilds *appguild.Service, availSvc *appavailability.Service, refresher *weekly.Refresher, welcome *WelcomeHandler, ownerID, supportURL string) *HelpConfigHandler {
	return &HelpConfigHandler{teams: teams, guilds: guilds, availSvc: availSvc, refresher: refresher, welcome: welcome, ownerID: ownerID, supportURL: supportURL}
}

func (h *HelpConfigHandler) CustomID() string {
	return discord.HelpPanelID
}

var _ discord.InteractionHandler = (*HelpConfigHandler)(nil)

func (h *HelpConfigHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := discord.LocaleMessages(i)
	if !discord.IsAdminOrOwner(i, h.ownerID) {
		return discord.RespondError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return discord.RespondError(s, i, m.ErrGuildOnly)
	}

	var customID string
	if i.Type == discordgo.InteractionModalSubmit {
		customID = i.ModalSubmitData().CustomID
	} else {
		customID = i.MessageComponentData().CustomID
	}
	action, teamID, step, err := discord.ParseHelpCustomID(customID)
	if err != nil {
		slog.WarnContext(ctx, "invalid help panel custom id", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrInvalid)
	}

	if lang, ok := strings.CutPrefix(action, discord.HelpActionLanguage); ok {
		return h.handleLanguage(ctx, s, i, lang)
	}
	switch action {
	case discord.HelpActionList:
		return h.showList(ctx, s, i, "")
	case discord.HelpActionCreate:
		return respondModal(s, i, discord.HelpCustomID(discord.HelpActionModalCreate, 0), m.CreateTitle, buildTeamNameField(m, ""))
	case discord.HelpActionModalCreate:
		return h.handleCreateSubmit(ctx, s, i)
	}

	t, err := h.teams.GetInGuild(ctx, i.GuildID, teamID)
	if errors.Is(err, team.ErrNotFound) {
		return h.showList(ctx, s, i, "⚠️ "+m.TeamGone)
	}
	if err != nil {
		slog.ErrorContext(ctx, "help config: get team", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	m = t.Messages()
	panelID := func(action string) string { return discord.HelpCustomID(action, t.ID) }

	switch action {
	case discord.HelpActionOpen:
		return showPanel(s, i, discord.BuildHelpPanel(t, ""))
	case discord.HelpActionOpenPublish:
		return respondModal(s, i, panelID(discord.HelpActionModalPublish), m.SectionPublish, buildPublishModalComponents(t))
	case discord.HelpActionOpenSchedule:
		return respondModal(s, i, panelID(discord.HelpActionModalSched), m.SectionSchedule, buildScheduleModalComponents(t))
	case discord.HelpActionOpenSlots:
		return respondModal(s, i, panelID(discord.HelpActionModalSlots), m.SectionSlots, buildSlotsModalComponents(t))
	case discord.HelpActionOpenMessage:
		return respondModal(s, i, panelID(discord.HelpActionModalMessage), m.SectionMessage, buildMessageModalComponents(t))
	case discord.HelpActionOpenRename:
		return respondModal(s, i, panelID(discord.HelpActionModalRename), m.RenameTitle, buildTeamNameField(m, t.Name))
	case discord.HelpActionOpenTimezone:
		// The panel turns into the picker, and back into the panel once a timezone is picked.
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: pickerStep(true, fmt.Sprintf(m.GuildPickerContent, m.TimezoneName(t.Location)), buildTimezonePicker(m, panelID(discord.HelpActionTimezone))),
		})
	case discord.HelpActionTimezone:
		return h.handleTimezonePick(ctx, s, i, t, step)
	case discord.HelpActionPublishNow:
		return h.handlePublishNow(ctx, s, i, t)
	case discord.HelpActionDelete:
		return showPanel(s, i, discord.BuildTeamDeletePanel(t))
	case discord.HelpActionConfirmDel:
		return h.handleDelete(ctx, s, i, t)
	}

	if i.Type != discordgo.InteractionModalSubmit {
		return discord.RespondError(s, i, m.ErrInvalid)
	}
	data := i.ModalSubmitData()
	switch action {
	case discord.HelpActionModalPublish:
		return h.handlePublishSubmit(ctx, s, i, t, data)
	case discord.HelpActionModalSched:
		return h.handleScheduleSubmit(ctx, s, i, t, data)
	case discord.HelpActionModalSlots:
		return h.handleSlotsSubmit(ctx, s, i, t, data)
	case discord.HelpActionModalMessage:
		return h.handleMessageSubmit(ctx, s, i, t, data)
	case discord.HelpActionModalRename:
		return h.handleRenameSubmit(ctx, s, i, t, data)
	}
	return discord.RespondError(s, i, m.ErrInvalid)
}

// respondModal opens a form.
func respondModal(s *discordgo.Session, i *discordgo.InteractionCreate, customID, title string, fields []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: customID, Title: title, Components: fields},
	})
}

// buildTeamNameField builds the team's name form, prefilled with its name when renaming.
func buildTeamNameField(m *i18n.Messages, name string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.Label{
			Label:       m.TeamNameLabel,
			Description: fmt.Sprintf(m.TeamNameHint, team.MaxNameLength),
			Component: discordgo.TextInput{
				CustomID:    "team_name",
				Style:       discordgo.TextInputShort,
				Placeholder: m.TeamNameSample,
				Value:       name,
				Required:    new(true),
				MinLength:   1,
				MaxLength:   team.MaxNameLength,
			},
		},
	}
}

// buildPublishModalComponents builds the channel and role form, prefilled with the current
// ones. Each can be picked or, by typing a name, created. The role is optional:
// unselecting it stops the mentions.
func buildPublishModalComponents(t *team.Team) []discordgo.MessageComponent {
	m := t.Messages()
	nameField := func(label, hint, customID, placeholder string) discordgo.Label {
		return discordgo.Label{
			Label:       label,
			Description: hint,
			Component: discordgo.TextInput{
				CustomID:    customID,
				Style:       discordgo.TextInputShort,
				Placeholder: placeholder,
				Required:    new(false),
				MaxLength:   100,
			},
		}
	}
	return []discordgo.MessageComponent{
		discordgo.Label{
			Label:       m.ChannelLabel,
			Description: m.ChannelHint,
			Component: discordgo.SelectMenu{
				CustomID:      "channel",
				MenuType:      discordgo.ChannelSelectMenu,
				ChannelTypes:  []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
				MinValues:     new(0),
				MaxValues:     1,
				Required:      new(false),
				DefaultValues: defaultChannelValues(t.ChannelID),
			},
		},
		nameField(m.NewChannelLabel, m.NewChannelHint, "new_channel", "dispos-"+t.Name),
		discordgo.Label{
			Label:       m.RoleLabel,
			Description: m.RoleHint,
			Component: discordgo.SelectMenu{
				CustomID:      "role",
				MenuType:      discordgo.RoleSelectMenu,
				MinValues:     new(0),
				MaxValues:     1,
				Required:      new(false),
				DefaultValues: defaultRoleValues(t.RoleID),
			},
		},
		nameField(m.NewRoleLabel, m.NewRoleHint, "new_role", t.Name),
	}
}

// buildScheduleModalComponents builds the publication form, prefilled with the current schedule.
func buildScheduleModalComponents(t *team.Team) []discordgo.MessageComponent {
	m := t.Messages()
	days := []discordgo.SelectMenuOption{
		{Label: m.PublishSunday, Value: strconv.Itoa(int(time.Sunday)), Default: t.Schedule.Weekday == time.Sunday},
		{Label: m.PublishMonday, Value: strconv.Itoa(int(time.Monday)), Default: t.Schedule.Weekday == time.Monday},
	}
	hours := make([]discordgo.SelectMenuOption, 24)
	for h := range hours {
		hours[h] = discordgo.SelectMenuOption{Label: m.HourLabel(h), Value: strconv.Itoa(h), Default: h == t.Schedule.Hour}
	}
	return []discordgo.MessageComponent{
		selectField(m.PublishDayLabel, m.PublishDayHint, "publish_weekday", true, days),
		selectField(m.PublishHourLabel, m.PublishHourHint, "publish_hour", true, hours),
	}
}

// Values of the slots form's member time select.
const (
	memberTimeServer = "server" // everyone sees the team's time
	memberTimeOwn    = "own"    // each member sees their own timezone
)

// buildSlotsModalComponents builds the slots form, prefilled with the current layout.
func buildSlotsModalComponents(t *team.Team) []discordgo.MessageComponent {
	m := t.Messages()
	current := t.SlotLayout
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
		{Label: m.MemberTimeServer, Value: memberTimeServer, Default: !t.MemberTimezones},
		{Label: m.MemberTimeOwn, Value: memberTimeOwn, Default: t.MemberTimezones},
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
func buildMessageModalComponents(t *team.Team) []discordgo.MessageComponent {
	m := t.Messages()
	languages := make([]discordgo.SelectMenuOption, len(i18n.Langs))
	for n, lang := range i18n.Langs {
		languages[n] = discordgo.SelectMenuOption{Label: m.LanguageNames[n], Value: string(lang), Default: lang == i18n.Parse(string(t.Language))}
	}
	displays := []discordgo.SelectMenuOption{
		{Label: m.DisplayClassic, Value: string(team.DisplayClassic), Default: t.Display != team.DisplayImage},
		{Label: m.DisplayImage, Value: string(team.DisplayImage), Default: t.Display == team.DisplayImage},
	}
	fullAt := []discordgo.SelectMenuOption{{Label: m.FullAtAuto, Value: "0", Default: t.FullAt <= 0}}
	for n := 1; n <= availability.MaxFullAt; n++ {
		fullAt = append(fullAt, discordgo.SelectMenuOption{Label: m.Players(n), Value: strconv.Itoa(n), Default: t.FullAt == n})
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

// handlePublishSubmit saves the channel and the role, created first when a name was typed:
// one that can't be created keeps its previous value and is reported, the rest is saved.
// A new channel (or the same one picked again for a paused team) gets the weekly message
// right away.
func (h *HelpConfigHandler) handlePublishSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *team.Team, data discordgo.ModalSubmitInteractionData) error {
	m := previous.Messages()
	channelID := cmp.Or(modalSelectValue(data.Components, "channel"), previous.ChannelID)
	roleID := modalSelectValue(data.Components, "role")
	newChannel := modalTextValue(data.Components, "new_channel")
	newRole := modalTextValue(data.Components, "new_role")

	answer := showPanel
	if newChannel != "" || newRole != "" {
		// Discord wants the answer within 3 seconds, and creating takes a few calls.
		if err := deferPanelUpdate(s, i); err != nil {
			return err
		}
		answer = editPanel
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), weekly.Timeout)
		defer cancel()
	}

	var warnings string
	warn := func(text string) { warnings += "\n⚠️ " + text }
	if newChannel != "" {
		created, err := s.GuildChannelCreateComplex(i.GuildID, discordgo.GuildChannelCreateData{Name: newChannel, Type: discordgo.ChannelTypeGuildText}, discordgo.WithContext(ctx))
		if err != nil {
			slog.ErrorContext(ctx, "help config: create channel", "guild", i.GuildID, "err", err)
			warn(creationError(err, m.NeedManageChannels, m.CreateChannelFailed))
			channelID = previous.ChannelID
		} else {
			channelID = created.ID
		}
	}
	if newRole != "" {
		created, err := s.GuildRoleCreate(i.GuildID, &discordgo.RoleParams{Name: newRole, Mentionable: new(true)}, discordgo.WithContext(ctx))
		if err != nil {
			slog.ErrorContext(ctx, "help config: create role", "guild", i.GuildID, "err", err)
			warn(creationError(err, m.NeedManageRoles, m.CreateRoleFailed))
			roleID = previous.RoleID
		} else {
			roleID = created.ID
		}
	}

	// A paused team picking its channel again resumes; one whose new channel couldn't be
	// created stays as it was.
	channelChanged := channelID != "" && (channelID != previous.ChannelID || (!previous.Active && newChannel == ""))
	roleChanged := roleID != previous.RoleID
	if !channelChanged && !roleChanged {
		if warnings != "" {
			return answer(s, i, discord.BuildHelpPanel(previous, warnings[1:]))
		}
		return answer(s, i, discord.BuildHelpPanel(previous, "✅ "+m.NothingChanged))
	}
	if channelChanged && newChannel == "" && !botCanPost(s, channelID) {
		warn(m.CantPostThere)
	}
	if roleChanged && newRole == "" {
		if role, err := s.State.Role(i.GuildID, roleID); err == nil && !role.Mentionable {
			warn(m.RoleNotMentionable)
		}
	}

	failed := func(what string, err error) error {
		slog.ErrorContext(ctx, "help config: set "+what, "guild", i.GuildID, "err", err)
		return answer(s, i, discord.BuildHelpPanel(previous, "⚠️ "+m.ErrGeneric))
	}
	if channelChanged {
		if err := h.teams.SetChannel(ctx, previous.ID, channelID); err != nil {
			return failed("channel", err)
		}
	}
	if roleChanged {
		if err := h.teams.SetRole(ctx, previous.ID, roleID); err != nil {
			return failed("role", err)
		}
	}

	var changes []any
	if channelChanged {
		changes = append(changes, "channel", channelName(s, channelID))
	}
	if roleChanged {
		changes = append(changes, "role", roleName(s, i.GuildID, roleID))
	}
	event := "settings changed"
	if previous.ChannelID == "" && channelID != "" {
		event = "team set up"
	}
	logSettings(ctx, i, previous, event, changes...)
	h.welcome.Refresh(s, i.GuildID)

	t := *previous
	t.ChannelID, t.RoleID, t.Active = channelID, roleID, t.Active || channelChanged
	message := "✅ " + m.ConfigSaved
	if channelChanged {
		message += m.PublishedNow
		weekly.PublishInBackground(ctx, s, h.availSvc, &t)
	} else if roleChanged {
		// Only the mention changes: redraw the message, without a new one pinging the role.
		h.refresher.Request(s, t.ID)
	}
	if t.IsComplete() && !previous.IsComplete() {
		message += fmt.Sprintf(m.TeamSetUp, t.Name)
	}
	return answer(s, i, discord.BuildHelpPanel(&t, message+warnings))
}

// creationError picks the text of a channel or role Discord refused to create.
func creationError(err error, missingPermission, otherwise string) string {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Message != nil && rest.Message.Code == discordgo.ErrCodeMissingPermissions {
		return missingPermission
	}
	return otherwise
}

// handleScheduleSubmit saves the publication schedule. A new schedule gets the weekly
// message right away when its publication time is already past.
func (h *HelpConfigHandler) handleScheduleSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *team.Team, data discordgo.ModalSubmitInteractionData) error {
	m := previous.Messages()
	weekday, errDay := strconv.Atoi(modalSelectValue(data.Components, "publish_weekday"))
	hour, errHour := strconv.Atoi(modalSelectValue(data.Components, "publish_hour"))
	schedule, err := availability.NewSchedule(time.Weekday(weekday), hour)
	if errDay != nil || errHour != nil || err != nil {
		return discord.RespondError(s, i, m.ErrInvalid)
	}
	if schedule == previous.Schedule {
		return h.refreshPanel(ctx, s, i, previous.ID, "✅ "+m.NothingChanged)
	}
	if err := h.teams.SetSchedule(ctx, previous.ID, schedule); err != nil {
		slog.ErrorContext(ctx, "help config: set schedule", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	logSettings(ctx, i, previous, "settings changed", "schedule", m.ScheduleSummary(schedule))

	t := *previous
	t.Schedule = schedule
	republish, err := weekly.Due(ctx, h.availSvc, &t)
	if err != nil {
		slog.ErrorContext(ctx, "help config: check weekly message", "guild", i.GuildID, "err", err)
	}
	return h.saved(ctx, s, i, &t, m.ConfigSaved, republish)
}

// handleSlotsSubmit saves the slot layout and whose time the members see. A new layout
// starts the week over, as slot indexes of the old one would be misread with it.
func (h *HelpConfigHandler) handleSlotsSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *team.Team, data discordgo.ModalSubmitInteractionData) error {
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
			return discord.RespondError(s, i, m.HourOrder)
		}
	}
	memberTimezones := modalSelectValue(data.Components, "member_time") == memberTimeOwn
	layoutChanged := layout != previous.SlotLayout
	if !layoutChanged && memberTimezones == previous.MemberTimezones {
		return h.refreshPanel(ctx, s, i, previous.ID, "✅ "+m.NothingChanged)
	}

	message := m.ConfigSaved
	if layoutChanged {
		if err := h.teams.SetMessageSettings(ctx, previous.ID, layout, previous.Display, previous.Language); err != nil {
			slog.ErrorContext(ctx, "help config: set slots", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
		if err := h.availSvc.ResetWeek(ctx, previous.ID, h.availSvc.Week(previous.Schedule, previous.Location)); err != nil {
			slog.ErrorContext(ctx, "help config: reset week", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
		message += m.SlotsReset
	}
	if memberTimezones != previous.MemberTimezones {
		if err := h.teams.SetMemberTimezones(ctx, previous.ID, memberTimezones); err != nil {
			slog.ErrorContext(ctx, "help config: set member timezones", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
	}

	var changes []any
	if layoutChanged {
		changes = append(changes, "slots", m.LayoutSummary(layout))
	}
	if memberTimezones != previous.MemberTimezones {
		changes = append(changes, "member_timezones", memberTimezones)
	}
	logSettings(ctx, i, previous, "settings changed", changes...)

	t := *previous
	t.SlotLayout, t.MemberTimezones = layout, memberTimezones
	// The message changes with the layout, or with the "Mon fuseau" button of hourly slots.
	republish := layoutChanged || t.MemberTime() != previous.MemberTime()
	return h.saved(ctx, s, i, &t, message, republish)
}

// handleMessageSubmit saves the display, the language and the green threshold. A new
// display or language posts the message again (it may switch between an embed and
// Components V2, which an edit can't do); a new threshold only redraws it.
func (h *HelpConfigHandler) handleMessageSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *team.Team, data discordgo.ModalSubmitInteractionData) error {
	lang := i18n.Parse(modalSelectValue(data.Components, "language"))
	m := i18n.For(lang) // answer in the language just picked
	display := team.DisplayClassic
	if modalSelectValue(data.Components, "display_style") == string(team.DisplayImage) {
		display = team.DisplayImage
	}
	fullAt, err := strconv.Atoi(modalSelectValue(data.Components, "full_at"))
	if err != nil || fullAt < 0 || fullAt > availability.MaxFullAt {
		return discord.RespondError(s, i, m.ErrInvalid)
	}
	republish := display != previous.Display || lang != previous.Language
	if !republish && fullAt == previous.FullAt {
		return h.refreshPanel(ctx, s, i, previous.ID, "✅ "+m.NothingChanged)
	}

	if republish {
		if err := h.teams.SetMessageSettings(ctx, previous.ID, previous.SlotLayout, display, lang); err != nil {
			slog.ErrorContext(ctx, "help config: set message settings", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
	}
	if fullAt != previous.FullAt {
		if err := h.teams.SetFullAt(ctx, previous.ID, fullAt); err != nil {
			slog.ErrorContext(ctx, "help config: set full at", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
		if !republish {
			// Only the colors change: redraw the message, without a new one pinging the role.
			h.refresher.Request(s, previous.ID)
		}
	}

	var changes []any
	if display != previous.Display {
		changes = append(changes, "display", display)
	}
	if lang != previous.Language {
		changes = append(changes, "language", lang)
	}
	if fullAt != previous.FullAt {
		changes = append(changes, "full_at", fullAt)
	}
	logSettings(ctx, i, previous, "settings changed", changes...)

	t := *previous
	t.Display, t.Language, t.FullAt = display, lang, fullAt
	return h.saved(ctx, s, i, &t, m.ConfigSaved, republish)
}

// handleTimezonePick saves the team's timezone picked in the picker. Hourly slots move
// with it, so the week's availabilities are reset as for a new slot layout.
func (h *HelpConfigHandler) handleTimezonePick(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, previous *team.Team, step string) error {
	m := previous.Messages()
	loc, err := readTimezonePick(ctx, s, i, m, discord.HelpCustomID(discord.HelpActionTimezone, previous.ID), step, m.GuildPickerCountry, true)
	if loc == nil {
		return err
	}
	if loc.String() == previous.Location.String() {
		return h.refreshPanel(ctx, s, i, previous.ID, "✅ "+m.NothingChanged)
	}
	if err := h.teams.SetTimezone(ctx, previous.ID, loc); err != nil {
		slog.ErrorContext(ctx, "help config: set timezone", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}

	logSettings(ctx, i, previous, "settings changed", "timezone", loc.String())

	message := fmt.Sprintf(m.GuildTimezoneSaved, m.TimezoneName(loc), m.Clock(time.Now().In(loc)))
	republish := previous.SlotLayout.IsHourly()
	if republish {
		if err := h.availSvc.ResetWeek(ctx, previous.ID, h.availSvc.Week(previous.Schedule, previous.Location)); err != nil {
			slog.ErrorContext(ctx, "help config: reset week", "guild", i.GuildID, "err", err)
			return discord.RespondError(s, i, m.ErrGeneric)
		}
		message += m.SlotsReset
	}

	t := *previous
	t.Location = loc
	// Half-day slots don't move: republish only when the current week changed.
	if !republish && t.ChannelID != "" {
		if republish, err = weekly.Due(ctx, h.availSvc, &t); err != nil {
			slog.ErrorContext(ctx, "help config: check weekly message", "guild", i.GuildID, "err", err)
		}
	}
	return h.saved(ctx, s, i, &t, message, republish)
}

// saved shows the panel again after a change, once the weekly message is being posted
// again when asked (in the background: Discord wants the answer within 3 seconds).
func (h *HelpConfigHandler) saved(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, t *team.Team, message string, republish bool) error {
	if republish && t.ChannelID != "" {
		message += t.Messages().PublishedNow
		weekly.PublishInBackground(ctx, s, h.availSvc, t)
	}
	return h.refreshPanel(ctx, s, i, t.ID, "✅ "+message)
}

// logSettings logs a change of the team's settings, with who made it.
func logSettings(ctx context.Context, i *discordgo.InteractionCreate, t *team.Team, event string, changes ...any) {
	slog.InfoContext(ctx, event, append([]any{"guild", i.GuildID, "team", t.Name, "by", discord.UserName(i)}, changes...)...)
}

// channelName returns "#name" of a channel the bot knows, else its ID.
func channelName(s *discordgo.Session, channelID string) string {
	if c, err := s.State.Channel(channelID); err == nil {
		return "#" + c.Name
	}
	return channelID
}

// roleName returns "@name" of a guild role the bot knows, else its ID ("none" for no role).
func roleName(s *discordgo.Session, guildID, roleID string) string {
	if roleID == "" {
		return "none"
	}
	if r, err := s.State.Role(guildID, roleID); err == nil {
		return "@" + r.Name
	}
	return roleID
}

// modalSelectValue returns the first selected value of a modal select menu, or "".
func modalSelectValue(components []discordgo.MessageComponent, customID string) string {
	var values []string
	switch c := findModalComponent(components, customID).(type) {
	case discordgo.SelectMenu:
		values = c.Values
	case *discordgo.SelectMenu:
		values = c.Values
	}
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// refreshPanel shows the team's panel again with the saved settings, in its language.
func (h *HelpConfigHandler) refreshPanel(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, teamID int64, status string) error {
	t, err := h.teams.GetInGuild(ctx, i.GuildID, teamID)
	if err != nil {
		slog.ErrorContext(ctx, "help config: get team", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, discord.LocaleMessages(i).ErrGeneric)
	}
	return showPanel(s, i, discord.BuildHelpPanel(t, status))
}

// showPanel replaces the panel the interaction comes from.
func showPanel(s *discordgo.Session, i *discordgo.InteractionCreate, components []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsIsComponentsV2, Components: components},
	})
}

// deferPanelUpdate acknowledges a panel interaction before slow Discord calls: the panel
// is then replaced with editPanel.
func deferPanelUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
}

// editPanel replaces the panel of an interaction acknowledged with deferPanelUpdate.
func editPanel(s *discordgo.Session, i *discordgo.InteractionCreate, components []discordgo.MessageComponent) error {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Flags: discordgo.MessageFlagsIsComponentsV2, Components: &components})
	return err
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

// findModalComponent returns the select menu or text input of a form, inside its action
// rows and labels.
func findModalComponent(components []discordgo.MessageComponent, customID string) discordgo.MessageComponent {
	for _, comp := range components {
		switch c := comp.(type) {
		case discordgo.SelectMenu:
			if c.CustomID == customID {
				return c
			}
		case *discordgo.SelectMenu:
			if c.CustomID == customID {
				return c
			}
		case discordgo.TextInput:
			if c.CustomID == customID {
				return c
			}
		case *discordgo.TextInput:
			if c.CustomID == customID {
				return c
			}
		case discordgo.ActionsRow:
			if found := findModalComponent(c.Components, customID); found != nil {
				return found
			}
		case *discordgo.ActionsRow:
			if found := findModalComponent(c.Components, customID); found != nil {
				return found
			}
		case discordgo.Label:
			if found := findModalComponent([]discordgo.MessageComponent{c.Component}, customID); found != nil {
				return found
			}
		case *discordgo.Label:
			if found := findModalComponent([]discordgo.MessageComponent{c.Component}, customID); found != nil {
				return found
			}
		}
	}
	return nil
}
