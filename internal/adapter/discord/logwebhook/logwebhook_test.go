package logwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestParseURL(t *testing.T) {
	id, token, err := parseURL("https://discord.com/api/webhooks/123/abc-DEF")
	if err != nil || id != "123" || token != "abc-DEF" {
		t.Fatalf("parseURL = %q, %q, %v", id, token, err)
	}
	for _, bad := range []string{"", "http://discord.com/api/webhooks/1/a", "https://discord.com/api/webhooks/1", "https://discord.com/channels/1/2"} {
		if _, _, err := parseURL(bad); err == nil {
			t.Errorf("parseURL(%q): want an error", bad)
		}
	}
}

// TestHandlerSendsBatch checks that records logged together reach the webhook in one
// message, labelled by level, with the guild named, and still reach the inner handler.
func TestHandlerSendsBatch(t *testing.T) {
	var mu sync.Mutex
	var messages []discordgo.WebhookParams
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p discordgo.WebhookParams
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("decode webhook body: %v", err)
		}
		mu.Lock()
		messages = append(messages, p)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	endpoint := discordgo.EndpointWebhookToken
	discordgo.EndpointWebhookToken = func(id, token string) string { return srv.URL + "/" + id + "/" + token }
	defer func() { discordgo.EndpointWebhookToken = endpoint }()

	var inner bytes.Buffer
	h, err := New("https://discord.com/api/webhooks/1/tok", slog.NewTextHandler(&inner, nil))
	if err != nil {
		t.Fatal(err)
	}
	h.SetGuildNames(func(id string) string { return map[string]string{"42": "Les Calamars"}[id] })
	log := slog.New(h)
	log.Info("guild set up", "guild", "42")
	log.Warn("lost access", "guild", "7")
	log.With("guild", "42").WithGroup("req").Error("command failed", "err", "boom")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.Close(ctx)

	mu.Lock()
	defer mu.Unlock()
	if len(messages) != 1 || len(messages[0].Embeds) != 3 {
		t.Fatalf("want one message of 3 embeds, got %+v", messages)
	}
	want := []struct{ title, description string }{
		{"[LOG] guild set up", "**guild** : Les Calamars (`42`)\n"},
		{"[WARNING] lost access", "**guild** : 7\n"},
		{"[ERROR] command failed", "**guild** : Les Calamars (`42`)\n**req.err** : boom\n"},
	}
	for n, e := range messages[0].Embeds {
		if e.Title != want[n].title || e.Description != want[n].description {
			t.Errorf("embed %d = %q %q, want %q %q", n, e.Title, e.Description, want[n].title, want[n].description)
		}
	}
	if messages[0].AllowedMentions == nil || len(messages[0].AllowedMentions.Parse) != 0 {
		t.Errorf("the logs must never ping: %+v", messages[0].AllowedMentions)
	}
	if got := strings.Count(inner.String(), "\n"); got != 3 {
		t.Errorf("inner handler got %d records, want 3:\n%s", got, inner.String())
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("héhé", 3); got != "hé…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abc", 3); got != "abc" {
		t.Errorf("truncate = %q", got)
	}
}
