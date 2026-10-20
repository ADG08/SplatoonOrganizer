package discord

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
)

// interactionTimeout bounds the work done for one interaction: Discord drops an
// interaction not answered within 3 seconds, so waiting longer on the database is useless.
const interactionTimeout = 3 * time.Second

// Bot wraps the Discord session and command registry.
type Bot struct {
	session  *discordgo.Session
	registry *Registry
}

// NewBot creates a new Discord bot.
func NewBot(token string, registry *Registry) (*Bot, error) {
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
	slog.Debug("Discord session opened")
	return nil
}

// Close closes the Discord connection.
func (b *Bot) Close() {
	if err := b.session.Close(); err != nil {
		slog.Error("close Discord session", "err", err)
		return
	}
	slog.Debug("Discord session closed")
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), interactionTimeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "panic in interaction handler", "guild", i.GuildID, "panic", r)
		}
	}()

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		cmd, ok := b.registry.Command(data.Name)
		if !ok {
			slog.WarnContext(ctx, "no command registered", "command", data.Name)
			return
		}
		if err := cmd.Execute(ctx, s, i); err != nil {
			slog.ErrorContext(ctx, "command failed", "command", data.Name, "guild", i.GuildID, "err", err)
		}
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		handler, ok := b.registry.Handler(data.CustomID)
		if !ok {
			slog.WarnContext(ctx, "no handler registered", "custom_id", data.CustomID)
			return
		}
		if err := handler.Handle(ctx, s, i); err != nil {
			slog.ErrorContext(ctx, "interaction failed", "custom_id", data.CustomID, "guild", i.GuildID, "err", err)
		}
	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		handler, ok := b.registry.Handler(data.CustomID)
		if !ok {
			slog.WarnContext(ctx, "no handler registered for modal", "custom_id", data.CustomID)
			return
		}
		if err := handler.Handle(ctx, s, i); err != nil {
			slog.ErrorContext(ctx, "modal failed", "custom_id", data.CustomID, "guild", i.GuildID, "err", err)
		}
	default:
		// ignore other interaction types for now
	}
}

// RegisterSlashCommands syncs the global slash commands with Discord using a bulk
// overwrite, which also removes the commands no longer in the registry.
func (b *Bot) RegisterSlashCommands(appID string) error {
	acs := make([]*discordgo.ApplicationCommand, 0, len(b.registry.commands))
	for _, c := range b.registry.commands {
		acs = append(acs, c.ApplicationCommand())
	}
	if _, err := b.session.ApplicationCommandBulkOverwrite(appID, "", acs); err != nil {
		return fmt.Errorf("bulk overwriting slash commands: %w", err)
	}
	return nil
}
