package discord

import (
	"context"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type handler string

func (h handler) CustomID() string { return string(h) }
func (h handler) Handle(context.Context, *discordgo.Session, *discordgo.InteractionCreate) error {
	return nil
}

func TestRegistryHandler(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"availability:tz", "availability:tzedit", "help:config"} {
		if err := r.RegisterHandler(handler(id)); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.RegisterHandler(handler("help:config")); err == nil {
		t.Error("registering an ID twice should fail")
	}

	for customID, want := range map[string]string{
		"availability:tz":          "availability:tz",
		"availability:tz:dispos:w": "availability:tz",
		"availability:tzedit:grid": "availability:tzedit", // not taken by "availability:tz"
		"help:config:open_publish": "help:config",
		"help:config:timezone:c":   "help:config",
	} {
		if h, ok := r.Handler(customID); !ok || h.CustomID() != want {
			t.Errorf("%s: got %v, want %s", customID, h, want)
		}
	}
	if _, ok := r.Handler("availability:unknown"); ok {
		t.Error("unknown custom ID matched a handler")
	}
}
