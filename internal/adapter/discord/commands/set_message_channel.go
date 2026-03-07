package commands

import (
	"context"
	"log"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/bwmarrin/discordgo"
)

const CommandNameSetMessageChannel = "set-message-channel"

// SetMessageChannelCommand handles the /set-message-channel slash command.
type SetMessageChannelCommand struct {
	svc *appguildconfig.Service
}

// NewSetMessageChannelCommand creates a new set-message-channel command.
func NewSetMessageChannelCommand(svc *appguildconfig.Service) *SetMessageChannelCommand {
	return &SetMessageChannelCommand{svc: svc}
}

func (c *SetMessageChannelCommand) Name() string {
	return CommandNameSetMessageChannel
}

func (c *SetMessageChannelCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	perm := int64(discordgo.PermissionAdministrator)
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameSetMessageChannel,
		Description:              "Définit le channel où poster le message des disponibilités",
		DefaultMemberPermissions: &perm,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "Channel où poster le message des dispos",
				Required:    true,
			},
		},
	}
}

var _ discord.Command = (*SetMessageChannelCommand)(nil)

func (c *SetMessageChannelCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	if guildID == "" {
		return respondWithError(s, i, "Cette commande doit être utilisée sur un serveur.")
	}

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return respondWithError(s, i, "Indique le channel à utiliser.")
	}
	channelOpt := opts[0]
	ch := channelOpt.ChannelValue(s)
	if ch == nil {
		return respondWithError(s, i, "Channel invalide.")
	}

	ctx := context.Background()
	if err := c.svc.SetChannel(ctx, guildID, ch.ID); err != nil {
		log.Printf("set-message-channel: %v", err)
		return respondWithError(s, i, "Erreur lors de l'enregistrement du channel.")
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Channel des dispos enregistré : <#" + ch.ID + ">",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
