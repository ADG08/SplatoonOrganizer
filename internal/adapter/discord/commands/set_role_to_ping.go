package commands

import (
	"context"
	"log"

	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/bwmarrin/discordgo"
)

const CommandNameSetRoleToPing = "set-role-to-ping"

// SetRoleToPingCommand handles the /set-role-to-ping slash command.
type SetRoleToPingCommand struct {
	svc     *appguildconfig.Service
	ownerID string
}

// NewSetRoleToPingCommand creates a new set-role-to-ping command. ownerID is the Discord user ID of the bot creator (optional).
func NewSetRoleToPingCommand(svc *appguildconfig.Service, ownerID string) *SetRoleToPingCommand {
	return &SetRoleToPingCommand{svc: svc, ownerID: ownerID}
}

func (c *SetRoleToPingCommand) Name() string {
	return CommandNameSetRoleToPing
}

func (c *SetRoleToPingCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNameSetRoleToPing,
		Description: "Définit le rôle à mentionner quand les dispos sont postées (admin ou créateur du bot)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "Rôle à ping lors du post des dispos",
				Required:    true,
			},
		},
	}
}

var _ discord.Command = (*SetRoleToPingCommand)(nil)

func (c *SetRoleToPingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if !isAdminOrOwner(i, c.ownerID) {
		return respondWithError(s, i, ErrAdminOrOwner)
	}
	guildID := i.GuildID
	if guildID == "" {
		return respondWithError(s, i, "Cette commande doit être utilisée sur un serveur.")
	}

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return respondWithError(s, i, "Indique le rôle à utiliser.")
	}
	roleOpt := opts[0]
	role := roleOpt.RoleValue(s, guildID)
	if role == nil {
		return respondWithError(s, i, "Rôle invalide.")
	}

	ctx := context.Background()
	if err := c.svc.SetRole(ctx, guildID, role.ID); err != nil {
		log.Printf("set-role-to-ping: %v", err)
		return respondWithError(s, i, "Erreur lors de l'enregistrement du rôle.")
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Rôle à ping enregistré : <@&" + role.ID + ">. Il sera mentionné à chaque post des dispos.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
