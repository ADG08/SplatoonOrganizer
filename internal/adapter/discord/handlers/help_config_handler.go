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
	if !isAdminOrOwnerInteraction(i, h.ownerID) {
		return respondWithError(s, i, ErrAdminOrOwnerInteraction)
	}

	guildID := i.GuildID
	if guildID == "" {
		return respondWithError(s, i, "Cette interaction doit etre utilisee sur un serveur.")
	}

	if i.Type == discordgo.InteractionModalSubmit {
		return h.handleModalSubmit(s, i, guildID)
	}

	data := i.MessageComponentData()
	switch data.CustomID {
	case discord.HelpPanelOpenChannelModalID:
		cfg, err := h.svc.Get(context.Background(), guildID)
		if err != nil {
			log.Printf("help config: get config before channel modal: %v", err)
		}
		defaultChannelID := ""
		if cfg != nil {
			defaultChannelID = cfg.ChannelID
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: discord.HelpPanelModalChannelID,
				Title:    "Configurer le channel",
				Components: []discordgo.MessageComponent{
					discordgo.Label{
						Label: "Channel des dispos",
						Component: discordgo.SelectMenu{
							CustomID:      "channel_selected",
							MenuType:      discordgo.ChannelSelectMenu,
							Placeholder:   "Choisis un channel texte",
							ChannelTypes:  []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
							MinValues:     intPtr(1),
							MaxValues:     1,
							Required:      boolPtr(true),
							DefaultValues: defaultChannelValues(defaultChannelID),
						},
					},
				},
			},
		})
	case discord.HelpPanelOpenRoleModalID:
		cfg, err := h.svc.Get(context.Background(), guildID)
		if err != nil {
			log.Printf("help config: get config before role modal: %v", err)
		}
		defaultRoleID := ""
		if cfg != nil {
			defaultRoleID = cfg.RoleID
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: discord.HelpPanelModalRoleID,
				Title:    "Configurer le role",
				Components: []discordgo.MessageComponent{
					discordgo.Label{
						Label: "Role a ping",
						Component: discordgo.SelectMenu{
							CustomID:      "roles_selected",
							MenuType:      discordgo.RoleSelectMenu,
							Placeholder:   "Choisis le role a mentionner",
							MinValues:     intPtr(1),
							MaxValues:     1,
							Required:      boolPtr(true),
							DefaultValues: defaultRoleValues(defaultRoleID),
						},
					},
				},
			},
		})
	case discord.HelpPanelOpenSlotsModalID:
		cfg, err := h.svc.Get(context.Background(), guildID)
		if err != nil {
			log.Printf("help config: get config before slots modal: %v", err)
			cfg = &guildconfig.GuildConfig{}
		}
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   discord.HelpPanelModalSlotsID,
				Title:      "Configurer les creneaux",
				Components: buildSlotsModalComponents(cfg),
			},
		})
	default:
		return respondWithError(s, i, "Interaction inconnue.")
	}
}

// buildSlotsModalComponents builds the slot layout and display form, prefilled with the current config.
func buildSlotsModalComponents(cfg *guildconfig.GuildConfig) []discordgo.MessageComponent {
	current := cfg.SlotLayout
	start, end := 13, 24
	if current.IsHourly() {
		start, end = current.StartHour, current.EndHour
	}

	modes := []discordgo.SelectMenuOption{
		{Label: "Apres-midi / Soir", Value: string(availability.ModeHalfDay), Default: !current.IsHourly()},
		{Label: "Par heure", Value: string(availability.ModeHourly), Default: current.IsHourly()},
	}
	// Discord allows 25 options per select menu: 0h..23h for the start, 1h..minuit for the end.
	var startOptions, endOptions []discordgo.SelectMenuOption
	for h := 0; h < 24; h++ {
		startOptions = append(startOptions, discordgo.SelectMenuOption{
			Label: availability.HourLabel(h), Value: strconv.Itoa(h), Default: h == start,
		})
		endOptions = append(endOptions, discordgo.SelectMenuOption{
			Label: availability.HourLabel(h + 1), Value: strconv.Itoa(h + 1), Default: h+1 == end,
		})
	}

	displays := []discordgo.SelectMenuOption{
		{Label: "Classique (texte)", Value: string(guildconfig.DisplayClassic), Default: cfg.Display != guildconfig.DisplayImage},
		{Label: "Image", Value: string(guildconfig.DisplayImage), Default: cfg.Display == guildconfig.DisplayImage},
	}

	return []discordgo.MessageComponent{
		discordgo.Label{
			Label:       "Affichage du message",
			Description: "Changer l'affichage republie le message de la semaine.",
			Component: discordgo.SelectMenu{
				CustomID:  "display_style",
				MinValues: intPtr(1),
				MaxValues: 1,
				Required:  boolPtr(true),
				Options:   displays,
			},
		},
		discordgo.Label{
			Label:       "Decoupage des journees",
			Description: "⚠️ Changer le decoupage reinitialise les dispos de la semaine en cours.",
			Component: discordgo.SelectMenu{
				CustomID:  "slot_mode",
				MinValues: intPtr(1),
				MaxValues: 1,
				Required:  boolPtr(true),
				Options:   modes,
			},
		},
		discordgo.Label{
			Label:       "Premiere heure (mode par heure)",
			Description: "Heure de Paris. Chaque membre verra les heures dans son fuseau.",
			Component: discordgo.SelectMenu{
				CustomID:  "slot_start_hour",
				MinValues: intPtr(1),
				MaxValues: 1,
				Required:  boolPtr(true),
				Options:   startOptions,
			},
		},
		discordgo.Label{
			Label: "Fin du dernier creneau (mode par heure)",
			Component: discordgo.SelectMenu{
				CustomID:  "slot_end_hour",
				MinValues: intPtr(1),
				MaxValues: 1,
				Required:  boolPtr(true),
				Options:   endOptions,
			},
		},
	}
}

func (h *HelpConfigHandler) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	data := i.ModalSubmitData()
	ctx := context.Background()

	switch data.CustomID {
	case discord.HelpPanelModalChannelID:
		menu := findModalSelectMenu(data.Components, "channel_selected")
		if menu == nil || len(menu.Values) == 0 {
			return respondWithError(s, i, "Aucun channel selectionne.")
		}
		channelID := menu.Values[0]
		if err := h.svc.SetChannel(ctx, guildID, channelID); err != nil {
			log.Printf("help config modal: set channel: %v", err)
			return respondWithError(s, i, "Erreur lors de l'enregistrement du channel.")
		}
	case discord.HelpPanelModalRoleID:
		menu := findModalSelectMenu(data.Components, "roles_selected")
		if menu == nil || len(menu.Values) == 0 {
			return respondWithError(s, i, "Aucun role selectionne.")
		}
		roleID := menu.Values[0]
		if err := h.svc.SetRole(ctx, guildID, roleID); err != nil {
			log.Printf("help config modal: set role: %v", err)
			return respondWithError(s, i, "Erreur lors de l'enregistrement du role.")
		}
	case discord.HelpPanelModalSlotsID:
		return h.handleSlotsSubmit(s, i, guildID, data)
	default:
		return respondWithError(s, i, "Modal inconnu.")
	}

	return h.refreshPanel(s, i, guildID, "Configuration enregistree via modal.")
}

func (h *HelpConfigHandler) handleSlotsSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, data discordgo.ModalSubmitInteractionData) error {
	ctx := context.Background()

	layout := availability.HalfDayLayout()
	if modalSelectValue(data.Components, "slot_mode") == string(availability.ModeHourly) {
		start, errStart := strconv.Atoi(modalSelectValue(data.Components, "slot_start_hour"))
		end, errEnd := strconv.Atoi(modalSelectValue(data.Components, "slot_end_hour"))
		if errStart != nil || errEnd != nil {
			return respondWithError(s, i, "Heures invalides.")
		}
		var err error
		if layout, err = availability.HourlyLayout(start, end); err != nil {
			return respondWithError(s, i, "La premiere heure doit etre avant la fin du dernier creneau.")
		}
	}

	display := guildconfig.DisplayClassic
	if modalSelectValue(data.Components, "display_style") == string(guildconfig.DisplayImage) {
		display = guildconfig.DisplayImage
	}

	previous, err := h.svc.Get(ctx, guildID)
	if err != nil {
		log.Printf("help config modal: get guild config: %v", err)
		return respondWithError(s, i, "Erreur lors de la recuperation de la configuration.")
	}
	layoutChanged := layout != previous.SlotLayout
	if !layoutChanged && display == previous.Display {
		return h.refreshPanel(s, i, guildID, "Rien n'a change.")
	}

	if err := h.svc.SetSlotSettings(ctx, guildID, layout, display); err != nil {
		log.Printf("help config modal: set slot settings: %v", err)
		return respondWithError(s, i, "Erreur lors de l'enregistrement des creneaux.")
	}

	message := "Affichage mis a jour : le message de la semaine a ete republie."
	if layoutChanged {
		// Slot indexes of the old layout would be misread with the new one: start the week over.
		if err := h.availSvc.ResetWeek(ctx, guildID, h.availSvc.CurrentWeek()); err != nil {
			log.Printf("help config modal: reset week: %v", err)
			return respondWithError(s, i, "Creneaux enregistres, mais impossible de reinitialiser les dispos de la semaine.")
		}
		message = "Creneaux mis a jour : les dispos de la semaine ont ete reinitialisees et le message republie."
	}

	// The message may switch between an embed and Components V2: post it again rather than edit it.
	cfg := previous
	cfg.SlotLayout, cfg.Display = layout, display
	if cfg.ChannelID == "" {
		message += " (aucun salon configure : rien n'a ete publie)"
	} else {
		go func() {
			if err := PublishWeeklyMessage(context.Background(), s, h.availSvc, cfg); err != nil {
				log.Printf("help config modal: republish weekly message: %v", err)
			}
		}()
	}

	return h.refreshPanel(s, i, guildID, message)
}

// modalSelectValue returns the first selected value of a modal select menu, or "".
func modalSelectValue(components []discordgo.MessageComponent, customID string) string {
	menu := findModalSelectMenu(components, customID)
	if menu == nil || len(menu.Values) == 0 {
		return ""
	}
	return menu.Values[0]
}

func (h *HelpConfigHandler) refreshPanel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID, description string) error {
	ctx := context.Background()
	cfg, err := h.svc.Get(ctx, guildID)
	if err != nil {
		log.Printf("help config: get guild config after update: %v", err)
		return respondWithError(s, i, "Configuration mise a jour, mais impossible de rafraichir l'affichage.")
	}

	embed := discord.BuildHelpPanelEmbed(
		"✅  Configuration mise a jour",
		description,
		discord.HelpPanelColorSuccess,
		cfg,
	)
	components := discord.BuildHelpPanelComponents()

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

const ErrAdminOrOwnerInteraction = "Tu dois etre administrateur du serveur ou le createur du bot pour utiliser ce panneau."
