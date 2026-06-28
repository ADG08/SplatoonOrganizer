package commands

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

const CommandNameAnnonce = "annonce"

// AnnonceCommand handles the /annonce slash command for publishing an embed announcement.
type AnnonceCommand struct {
	svc            *appguildconfig.Service
	ownerID        string
	defaultChannel string
}

// NewAnnonceCommand creates a new annonce command. ownerID and defaultChannelID are optional.
func NewAnnonceCommand(svc *appguildconfig.Service, ownerID, defaultChannelID string) *AnnonceCommand {
	return &AnnonceCommand{svc: svc, ownerID: ownerID, defaultChannel: defaultChannelID}
}

func (c *AnnonceCommand) Name() string {
	return CommandNameAnnonce
}

func (c *AnnonceCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNameAnnonce,
		Description: "Publier une annonce (embed) dans le salon des dispos",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "titre",
				Description: "Titre de l'annonce",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "Contenu de l'annonce (utilise \\n pour un retour à la ligne)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "Rôle à mentionner (optionnel)",
				Required:    false,
			},
		},
	}
}

var _ discord.Command = (*AnnonceCommand)(nil)

func (c *AnnonceCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if !isAdminOrOwner(i, c.ownerID) {
		return respondWithError(s, i, ErrAdminOrOwner)
	}
	guildID := i.GuildID
	if guildID == "" {
		return respondWithError(s, i, "Cette commande doit être utilisée sur un serveur.")
	}

	ctx := context.Background()
	cfg, err := c.svc.Get(ctx, guildID)
	if err != nil {
		log.Printf("annonce: get guild config: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de la configuration.")
	}

	channelID := cfg.ChannelID
	if channelID == "" {
		channelID = c.defaultChannel
	}
	if channelID == "" {
		return respondWithError(s, i, "Aucun salon configuré. Utilise `/help` pour définir le salon des dispos.")
	}

	opts := newOptionMap(i.ApplicationCommandData().Options)
	titre := strings.TrimSpace(opts.str("titre"))
	message := opts.str("message")
	roleID := opts.roleID("role")

	if titre == "" || strings.TrimSpace(message) == "" {
		return respondWithError(s, i, "Le titre et le message ne peuvent pas être vides.")
	}

	// Allow line breaks via the literal "\n" typed in the slash command option.
	message = strings.ReplaceAll(message, "\\n", "\n")

	description := message
	if i.Member != nil && i.Member.User != nil {
		// A mention pill is clickable and opens the Discord profile card (embed mentions never ping).
		description += "\n\n**Annonce de** <@" + i.Member.User.ID + ">"
	}

	embed := &discordgo.MessageEmbed{
		Title:       titre,
		Description: description,
		Color:       discord.HelpPanelColorDefault,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	send := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}}
	if roleID != "" {
		send.Content = "<@&" + roleID + ">"
		send.AllowedMentions = &discordgo.MessageAllowedMentions{
			Roles: []string{roleID},
		}
	}

	if _, err := s.ChannelMessageSendComplex(channelID, send); err != nil {
		log.Printf("annonce: send message: %v", err)
		return respondWithError(s, i, "Erreur lors de l'envoi de l'annonce.")
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Annonce publiée dans <#" + channelID + ">.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// optionMap indexes slash command options by name for convenient lookup.
type optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption

func newOptionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) optionMap {
	m := make(optionMap, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

func (m optionMap) str(name string) string {
	if o, ok := m[name]; ok {
		return o.StringValue()
	}
	return ""
}

// roleID returns the selected role ID for a Role option, or "" if absent.
func (m optionMap) roleID(name string) string {
	if o, ok := m[name]; ok {
		if id, ok := o.Value.(string); ok {
			return id
		}
	}
	return ""
}
