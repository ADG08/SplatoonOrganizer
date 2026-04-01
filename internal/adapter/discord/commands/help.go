package commands

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/bwmarrin/discordgo"
)

const CommandNameHelp = "help"

// HelpCommand handles the /help slash command for admin configuration.
type HelpCommand struct {
	svc     *appguildconfig.Service
	ownerID string
}

// NewHelpCommand creates a new help command. ownerID is optional.
func NewHelpCommand(svc *appguildconfig.Service, ownerID string) *HelpCommand {
	return &HelpCommand{svc: svc, ownerID: ownerID}
}

func (c *HelpCommand) Name() string {
	return CommandNameHelp
}

func (c *HelpCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNameHelp,
		Description: "Panneau admin pour configurer le bot",
	}
}

var _ discord.Command = (*HelpCommand)(nil)

func (c *HelpCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if !isAdminOrOwner(i, c.ownerID) {
		return respondWithError(s, i, ErrAdminOrOwner)
	}
	guildID := i.GuildID
	if guildID == "" {
		return respondWithError(s, i, "Cette commande doit etre utilisee sur un serveur.")
	}

	ctx := context.Background()
	cfg, err := c.svc.Get(ctx, guildID)
	if err != nil {
		log.Printf("help: get guild config: %v", err)
		return respondWithError(s, i, "Erreur lors de la recuperation de la configuration.")
	}

	embed := discord.BuildHelpPanelEmbed(
		"⚙️  Panneau de configuration",
		"Gerez les parametres du bot pour ce serveur.",
		discord.HelpPanelColorDefault,
		cfg.ChannelID,
		cfg.RoleID,
	)
	components := discord.BuildHelpPanelComponents()

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}
