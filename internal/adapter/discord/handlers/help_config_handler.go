package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const HelpConfigCustomIDPrefix = "help:config"

// HelpConfigHandler handles channel/role/slots config from /help panel.
type HelpConfigHandler struct {
	svc      *appguildconfig.Service
	availSvc *appavailability.Service
	ownerID  string
}

// NewHelpConfigHandler creates a new help config handler. ownerID is optional.
func NewHelpConfigHandler(svc *appguildconfig.Service, availSvc *appavailability.Service, ownerID string) *HelpConfigHandler {
	return &HelpConfigHandler{svc: svc, availSvc: availSvc, ownerID: ownerID}
}

func (h *HelpConfigHandler) CustomID() string {
	return HelpConfigCustomIDPrefix
}

var _ discord.InteractionHandler = (*HelpConfigHandler)(nil)

func (h *HelpConfigHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg := loadGuildConfig(s, i, h.svc)
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
		return h.handleModalSubmit(s, i, cfg)
	}

	switch i.MessageComponentData().CustomID {
	case discord.HelpPanelOpenChannelModalID:
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: discord.HelpPanelModalChannelID,
				Title:    m.ChannelModalTitle,
				Components: []discordgo.MessageComponent{
					discordgo.Label{
						Label: m.ChannelLabel,
						Component: discordgo.SelectMenu{
							CustomID:      "channel_selected",
							MenuType:      discordgo.ChannelSelectMenu,
							Placeholder:   m.ChannelHint,
							ChannelTypes:  []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
							MinValues:     intPtr(1),
							MaxValues:     1,
							Required:      boolPtr(true),
							DefaultValues: defaultChannelValues(cfg.ChannelID),
						},
					},
				},
			},
		})
	case discord.HelpPanelOpenRoleModalID:
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: discord.HelpPanelModalRoleID,
				Title:    m.RoleModalTitle,
				Components: []discordgo.MessageComponent{
					discordgo.Label{
						Label: m.RoleLabel,
						Component: discordgo.SelectMenu{
							CustomID:      "roles_selected",
							MenuType:      discordgo.RoleSelectMenu,
							Placeholder:   m.RoleHint,
							MinValues:     intPtr(1),
							MaxValues:     1,
							Required:      boolPtr(true),
							DefaultValues: defaultRoleValues(cfg.RoleID),
						},
					},
				},
			},
		})
	case discord.HelpPanelOpenSlotsModalID:
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   discord.HelpPanelModalSlotsID,
				Title:      m.MessageModalTitle,
				Components: buildSlotsModalComponents(cfg),
			},
		})
	default:
		return respondWithError(s, i, m.ErrInvalid)
	}
}

// buildSlotsModalComponents builds the language, display and slots form, prefilled with
// the current config (Discord allows 5 fields per form).
func buildSlotsModalComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	m := cfg.Messages()
	current := cfg.SlotLayout
	start, end := availability.DefaultStartHour, availability.DefaultEndHour
	if current.IsHourly() {
		start, end = current.StartHour, current.EndHour
	}

	languages := make([]discordgo.SelectMenuOption, len(i18n.Langs))
	for n, lang := range i18n.Langs {
		languages[n] = discordgo.SelectMenuOption{Label: m.LanguageNames[n], Value: string(lang), Default: lang == i18n.Parse(string(cfg.Language))}
	}
	displays := []discordgo.SelectMenuOption{
		{Label: m.DisplayClassic, Value: string(guildconfig.DisplayClassic), Default: cfg.Display != guildconfig.DisplayImage},
		{Label: m.DisplayImage, Value: string(guildconfig.DisplayImage), Default: cfg.Display == guildconfig.DisplayImage},
	}
	modes := []discordgo.SelectMenuOption{
		{Label: m.LayoutSummary(availability.HalfDayLayout()), Value: string(availability.ModeHalfDay), Default: !current.IsHourly()},
		{Label: m.ModeHourly, Value: string(availability.ModeHourly), Default: current.IsHourly()},
	}
	// Discord allows 25 options per select menu: 0h..23h for the start, 1h..minuit for the end.
	var startOptions, endOptions []discordgo.SelectMenuOption
	for h := 0; h < 24; h++ {
		startOptions = append(startOptions, discordgo.SelectMenuOption{
			Label: m.HourLabel(h), Value: strconv.Itoa(h), Default: h == start,
		})
		endOptions = append(endOptions, discordgo.SelectMenuOption{
			Label: m.HourLabel(h + 1), Value: strconv.Itoa(h + 1), Default: h+1 == end,
		})
	}

	selectField := func(label, hint, customID string, required bool, options []discordgo.SelectMenuOption) discordgo.Label {
		minValues := 0
		if required {
			minValues = 1
		}
		return discordgo.Label{
			Label:       label,
			Description: hint,
			Component: discordgo.SelectMenu{
				CustomID:  customID,
				MinValues: intPtr(minValues),
				MaxValues: 1,
				Required:  boolPtr(required),
				Options:   options,
			},
		}
	}
	return []discordgo.MessageComponent{
		selectField(m.LanguageLabel, "", "language", true, languages),
		selectField(m.DisplayLabel, m.DisplayHint, "display_style", true, displays),
		selectField(m.ModeLabel, m.ModeHint, "slot_mode", true, modes),
		// The hours are optional: they don't apply to "Après-midi / Soir".
		selectField(m.StartHourLabel, m.StartHourHint, "slot_start_hour", false, startOptions),
		selectField(m.EndHourLabel, m.EndHourHint, "slot_end_hour", false, endOptions),
	}
}

func (h *HelpConfigHandler) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *guildconfig.GuildConfig) error {
	data := i.ModalSubmitData()
	ctx := context.Background()
	m := cfg.Messages()

	switch data.CustomID {
	case discord.HelpPanelModalChannelID:
		channelID := modalSelectValue(data.Components, "channel_selected")
		if channelID == "" {
			return respondWithError(s, i, m.NothingSelected)
		}
		if err := h.svc.SetChannel(ctx, cfg.GuildID, channelID); err != nil {
			log.Printf("help config modal: set channel: %v", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	case discord.HelpPanelModalRoleID:
		roleID := modalSelectValue(data.Components, "roles_selected")
		if roleID == "" {
			return respondWithError(s, i, m.NothingSelected)
		}
		if err := h.svc.SetRole(ctx, cfg.GuildID, roleID); err != nil {
			log.Printf("help config modal: set role: %v", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
	case discord.HelpPanelModalSlotsID:
		return h.handleSlotsSubmit(s, i, cfg, data)
	default:
		return respondWithError(s, i, m.ErrInvalid)
	}

	return h.refreshPanel(s, i, cfg.GuildID, m.ConfigSaved)
}

func (h *HelpConfigHandler) handleSlotsSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, previous *guildconfig.GuildConfig, data discordgo.ModalSubmitInteractionData) error {
	ctx := context.Background()
	lang := i18n.Parse(modalSelectValue(data.Components, "language"))
	m := i18n.For(lang) // answer in the language just picked

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

	display := guildconfig.DisplayClassic
	if modalSelectValue(data.Components, "display_style") == string(guildconfig.DisplayImage) {
		display = guildconfig.DisplayImage
	}

	layoutChanged := layout != previous.SlotLayout
	if !layoutChanged && display == previous.Display && lang == previous.Language {
		return h.refreshPanel(s, i, previous.GuildID, m.NothingChanged)
	}

	if err := h.svc.SetMessageSettings(ctx, previous.GuildID, layout, display, lang); err != nil {
		log.Printf("help config modal: set message settings: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}

	message := m.Republished
	if layoutChanged {
		// Slot indexes of the old layout would be misread with the new one: start the week over.
		if err := h.availSvc.ResetWeek(ctx, previous.GuildID, h.availSvc.CurrentWeek()); err != nil {
			log.Printf("help config modal: reset week: %v", err)
			return respondWithError(s, i, m.ErrGeneric)
		}
		message = m.SlotsReset
	}

	// The message may switch between an embed and Components V2: post it again rather than edit it.
	cfg := *previous
	cfg.SlotLayout, cfg.Display, cfg.Language = layout, display, lang
	if cfg.ChannelID == "" {
		message += m.NotPublished
	} else {
		go func() {
			if err := PublishWeeklyMessage(context.Background(), s, h.availSvc, &cfg); err != nil {
				log.Printf("help config modal: republish weekly message: %v", err)
			}
		}()
	}

	return h.refreshPanel(s, i, previous.GuildID, message)
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
func (h *HelpConfigHandler) refreshPanel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, description string) error {
	cfg := loadGuildConfig(s, i, h.svc)
	if cfg == nil {
		return nil
	}
	embed := discord.BuildHelpPanelEmbed(cfg.Messages().PanelUpdated, description, discord.HelpPanelColorSuccess, cfg)
	components := discord.BuildHelpPanelComponents(cfg)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
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
