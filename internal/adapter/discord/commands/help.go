package commands

import (
	"context"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
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
	desc, descEN := description(func(m *i18n.Messages) string { return m.HelpDescription })
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameHelp,
		Description:              desc,
		DescriptionLocalizations: descEN,
	}
}

var _ discord.Command = (*HelpCommand)(nil)

func (c *HelpCommand) Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg, err := c.svc.Get(ctx, i.GuildID)
	if err != nil {
		log.Printf("help: get guild config: %v", err)
		return respondWithError(s, i, i18n.For(i18n.FR).ErrGeneric)
	}
	m := cfg.Messages()
	if !isAdminOrOwner(i, c.ownerID) {
		return respondWithError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return respondWithError(s, i, m.ErrGuildOnly)
	}

	embed := discord.BuildHelpPanelEmbed(m.PanelTitle, m.PanelDescription, discord.HelpPanelColorDefault, cfg)
	components := discord.BuildHelpPanelComponents(cfg)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral,
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}
