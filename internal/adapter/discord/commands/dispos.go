package commands

import (
	"context"
	"log"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/bwmarrin/discordgo"
)

const CommandNameDispos = "dispos"

// DisposCommand handles the /dispos slash command.
type DisposCommand struct {
	svc *appavailability.Service
}

// NewDisposCommand creates a new dispos command.
func NewDisposCommand(svc *appavailability.Service) *DisposCommand {
	return &DisposCommand{svc: svc}
}

func (c *DisposCommand) Name() string {
	return CommandNameDispos
}

func (c *DisposCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNameDispos,
		Description: "Affiche les disponibilités par créneau (visible par toi uniquement)",
	}
}

var _ discord.Command = (*DisposCommand)(nil)

func (c *DisposCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx := context.Background()
	week := c.svc.CurrentWeek()

	users, err := c.svc.GetAvailabilityUsers(ctx, week)
	if err != nil {
		log.Printf("error getting availability users: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération des disponibilités.")
	}

	description := c.svc.FormatUsersBySlot(users)
	embed := &discordgo.MessageEmbed{
		Title:       "Disponibilités par créneau",
		Description: description,
		Color:       0xFEE75C,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Semaine " + string(week) + " • Visible par toi uniquement"},
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
