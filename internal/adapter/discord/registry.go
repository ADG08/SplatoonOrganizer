package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Command defines a Discord slash command.
type Command interface {
	Name() string
	ApplicationCommand() *discordgo.ApplicationCommand
	Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// InteractionHandler handles Discord message component interactions.
type InteractionHandler interface {
	CustomID() string
	Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// Registry holds commands and handlers.
type Registry struct {
	commands map[string]Command
	handlers map[string]InteractionHandler
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
		handlers: make(map[string]InteractionHandler),
	}
}

// RegisterCommand registers a slash command.
func (r *Registry) RegisterCommand(cmd Command) error {
	name := cmd.Name()
	if name == "" {
		return fmt.Errorf("command name is empty")
	}
	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command %q already registered", name)
	}
	r.commands[name] = cmd
	return nil
}

// Command returns a command by name.
func (r *Registry) Command(name string) (Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// RegisterHandler registers an interaction handler.
func (r *Registry) RegisterHandler(h InteractionHandler) error {
	id := h.CustomID()
	if id == "" {
		return fmt.Errorf("handler CustomID is empty")
	}
	if _, exists := r.handlers[id]; exists {
		return fmt.Errorf("handler %q already registered", id)
	}
	r.handlers[id] = h
	return nil
}

// Handler returns the handler of a custom ID: the one registered with that ID, or with
// its prefix before a ":" ("availability:select" handles "availability:select:3").
func (r *Registry) Handler(customID string) (InteractionHandler, bool) {
	if h, ok := r.handlers[customID]; ok {
		return h, true
	}
	for id, h := range r.handlers {
		if strings.HasPrefix(customID, id+":") {
			return h, true
		}
	}
	return nil, false
}
