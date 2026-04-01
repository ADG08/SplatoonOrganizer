package discord

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Bot wraps the Discord session and command registry.
type Bot struct {
	session  *discordgo.Session
	registry *Registry
}

// NewBot creates a new Discord bot.
func NewBot(token string, registry *Registry) (*Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is empty")
	}
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating discord session: %w", err)
	}

	b := &Bot{
		session:  s,
		registry: registry,
	}

	s.AddHandler(b.onInteractionCreate)

	return b, nil
}

// Session returns the underlying Discord session.
func (b *Bot) Session() *discordgo.Session {
	return b.session
}

// Open opens the Discord WebSocket connection.
func (b *Bot) Open() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("opening discord session: %w", err)
	}
	log.Println("Discord session opened")
	return nil
}

// Close closes the Discord connection.
func (b *Bot) Close() error {
	if b.session == nil {
		return nil
	}
	b.session.Close()
	log.Println("Discord session closed")
	return nil
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in interaction handler: %v", r)
		}
	}()

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		cmd, ok := b.registry.GetCommand(data.Name)
		if !ok {
			log.Printf("no command registered for %q", data.Name)
			return
		}
		if err := cmd.Execute(s, i); err != nil {
			log.Printf("error executing command %q: %v", data.Name, err)
		}
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		handler, ok := b.registry.GetHandler(data.CustomID)
		if !ok {
			log.Printf("no handler registered for %q", data.CustomID)
			return
		}
		if err := handler.Handle(s, i); err != nil {
			log.Printf("error handling interaction %q: %v", data.CustomID, err)
		}
	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		handler, ok := b.registry.GetHandler(data.CustomID)
		if !ok {
			log.Printf("no handler registered for modal %q", data.CustomID)
			return
		}
		if err := handler.Handle(s, i); err != nil {
			log.Printf("error handling modal %q: %v", data.CustomID, err)
		}
	default:
		// ignore other interaction types for now
	}
}

// RegisterSlashCommands syncs slash commands with Discord using a bulk overwrite,
// which removes any previously registered commands that are no longer in the registry.
func (b *Bot) RegisterSlashCommands(appID, guildID string) error {
	if appID == "" {
		return fmt.Errorf("application ID is empty")
	}
	if guildID == "" {
		return fmt.Errorf("guild ID is empty")
	}

	var acs []*discordgo.ApplicationCommand
	for _, c := range b.registry.commands {
		ac := c.ApplicationCommand()
		if ac == nil {
			ac = &discordgo.ApplicationCommand{
				Name:        c.Name(),
				Description: "Splatoon organizer command",
			}
		}
		acs = append(acs, ac)
	}

	if _, err := b.session.ApplicationCommandBulkOverwrite(appID, guildID, acs); err != nil {
		return fmt.Errorf("bulk overwriting slash commands: %w", err)
	}

	return nil
}
