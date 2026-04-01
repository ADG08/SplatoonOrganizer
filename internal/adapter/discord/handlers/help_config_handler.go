package handlers

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

const HelpConfigCustomIDPrefix = "help:config"

// HelpConfigHandler handles channel/role config selects from /help panel.
type HelpConfigHandler struct {
	svc     *appguildconfig.Service
	ownerID string
}

// NewHelpConfigHandler creates a new help config handler.
func NewHelpConfigHandler(svc *appguildconfig.Service, ownerID string) *HelpConfigHandler {
	return &HelpConfigHandler{svc: svc, ownerID: ownerID}
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
	default:
		return respondWithError(s, i, "Interaction inconnue.")
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
	default:
		return respondWithError(s, i, "Modal inconnu.")
	}

	return h.refreshPanel(s, i, guildID, "Configuration enregistree via modal.")
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
		cfg.ChannelID,
		cfg.RoleID,
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
