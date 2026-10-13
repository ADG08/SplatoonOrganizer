package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Command defines a Discord slash command.
type Command interface {
	Name() string
	ApplicationCommand() *discordgo.ApplicationCommand
	Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// InteractionHandler handles Discord message component interactions.
type InteractionHandler interface {
	CustomID() string
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
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
	if cmd == nil {
		return fmt.Errorf("command is nil")
	}
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

// GetCommand returns a command by name.
func (r *Registry) GetCommand(name string) (Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// RegisterHandler registers an interaction handler.
func (r *Registry) RegisterHandler(h InteractionHandler) error {
	if h == nil {
		return fmt.Errorf("handler is nil")
	}
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

// GetHandler returns a handler by custom ID (supports prefix matching).
func (r *Registry) GetHandler(customID string) (InteractionHandler, bool) {
	if h, ok := r.handlers[customID]; ok {
		return h, true
	}
	for id, h := range r.handlers {
		if strings.HasPrefix(customID, id+":") {
			return h, true
		}
		if strings.HasPrefix(customID, id+"_") {
			return h, true
		}
	}
	return nil, false
}
