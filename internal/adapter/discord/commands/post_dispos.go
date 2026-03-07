package commands

import (
	"context"
	"errors"
	"log"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/scheduler"
	"github.com/bwmarrin/discordgo"
)

const CommandNamePostDispos = "post-dispos"

// PostDisposCommand handles the /post-dispos slash command.
type PostDisposCommand struct {
	sched   *scheduler.Scheduler
	ownerID string
}

// NewPostDisposCommand creates a new post-dispos command. ownerID is the Discord user ID of the bot creator (optional).
func NewPostDisposCommand(sched *scheduler.Scheduler, ownerID string) *PostDisposCommand {
	return &PostDisposCommand{sched: sched, ownerID: ownerID}
}

func (c *PostDisposCommand) Name() string {
	return CommandNamePostDispos
}

func (c *PostDisposCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNamePostDispos,
		Description: "Poste le message des disponibilités de la semaine (admin ou créateur du bot)",
	}
}

var _ discord.Command = (*PostDisposCommand)(nil)

func (c *PostDisposCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if !isAdminOrOwner(i, c.ownerID) {
		return respondWithError(s, i, ErrAdminOrOwner)
	}
	// Respond quickly to avoid Discord "application did not respond" timeouts.
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Je poste le message des disponibilités dans le channel configuré...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		return err
	}

	// Run the job asynchronously; errors are sent back to the user.
	go func() {
		ctx := context.Background()
		if err := c.sched.RunWeeklyNow(ctx); err != nil {
			log.Printf("post-dispos failed: %v", err)
			msg := "Une erreur s'est produite."
			if errors.Is(err, scheduler.ErrChannelNotConfigured) {
				msg = "Aucun channel configuré. Définis-le avec la commande **/set-message-channel**."
			}
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
	}()

	return nil
}
