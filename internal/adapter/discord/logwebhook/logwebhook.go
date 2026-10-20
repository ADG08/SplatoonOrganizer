// Package logwebhook copies the bot's logs into a Discord channel through a webhook: its
// owner sees in their own server the guilds joining, the settings changed, the warnings
// and the errors, without opening the hosting's logs.
package logwebhook

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

const (
	queueSize   = 100              // entries waiting to be sent; more are dropped
	batchDelay  = time.Second      // entries logged together are sent in one message
	maxEmbeds   = 10               // per webhook message
	maxChars    = 5500             // per webhook message: Discord allows 6000 in its embeds
	maxValue    = 400              // runes per attribute value
	sendTimeout = 10 * time.Second // per webhook message
	colorLog    = 0x5865F2         // blurple
	colorWarn   = 0xFEE75C         // yellow
	colorError  = 0xED4245         // red
)

// Handler is a slog.Handler sending the records to a Discord webhook, then to inner.
// Sending happens in the background: logging never waits for Discord, and records
// logged while Discord is slow or down are dropped past a queue, not retried forever.
type Handler struct {
	inner  slog.Handler
	attrs  []slog.Attr // from WithAttrs
	prefix string      // from WithGroup: "group."
	sink   *sink
}

type entry struct {
	level slog.Level
	time  time.Time
	msg   string
	attrs []slog.Attr
}

// sink is the webhook and its queue, shared by the handlers derived from New's.
type sink struct {
	session      *discordgo.Session // without token: only its webhook calls and rate limiting are used
	webhookID    string
	webhookToken string
	errLog       *slog.Logger // the sink's own errors: never sent to Discord, they would loop

	entries   chan entry
	dropped   atomic.Int64
	guildName atomic.Pointer[func(guildID string) string]
	stop      chan struct{}
	done      chan struct{}
}

// New returns a handler sending the records to the webhook URL (copied from the channel's
// Integrations settings), then to inner. Close sends what is left.
func New(webhookURL string, inner slog.Handler) (*Handler, error) {
	id, token, err := parseURL(webhookURL)
	if err != nil {
		return nil, err
	}
	session, err := discordgo.New("")
	if err != nil {
		return nil, fmt.Errorf("creating webhook session: %w", err)
	}
	s := &sink{
		session:      session,
		webhookID:    id,
		webhookToken: token,
		errLog:       slog.New(inner),
		entries:      make(chan entry, queueSize),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	go s.run()
	return &Handler{inner: inner, sink: s}, nil
}

// parseURL extracts the webhook's ID and token from https://discord.com/api/webhooks/<id>/<token>.
func parseURL(webhookURL string) (id, token string, err error) {
	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid webhook URL: %w", err)
	}
	_, rest, ok := strings.Cut(u.Path, "/webhooks/")
	id, token, _ = strings.Cut(rest, "/")
	if u.Scheme != "https" || !ok || id == "" || token == "" || strings.Contains(token, "/") {
		return "", "", fmt.Errorf("invalid webhook URL: want https://discord.com/api/webhooks/<id>/<token>")
	}
	return id, token, nil
}

// SetGuildNames gives the function naming the guilds of the "guild" attributes, which
// are IDs. It is set once the Discord session exists; until then the IDs alone are shown.
// A nil handler (no webhook configured) ignores it.
func (h *Handler) SetGuildNames(name func(guildID string) string) {
	if h != nil {
		h.sink.guildName.Store(&name)
	}
}

// Close sends the records still queued, waiting at most until ctx is done.
// A nil handler (no webhook configured) has nothing to send.
func (h *Handler) Close(ctx context.Context) {
	if h == nil {
		return
	}
	close(h.sink.stop)
	select {
	case <-h.sink.done:
	case <-ctx.Done():
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	attrs := make([]slog.Attr, 0, len(h.attrs)+r.NumAttrs())
	attrs = append(attrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, h.prefixed(a))
		return true
	})
	select {
	case h.sink.entries <- entry{level: r.Level, time: r.Time, msg: r.Message, attrs: attrs}:
	default:
		h.sink.dropped.Add(1)
	}
	return h.inner.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *h
	c.inner = h.inner.WithAttrs(attrs)
	c.attrs = append(c.attrs[:len(c.attrs):len(c.attrs)], attrs...)
	for n := len(h.attrs); n < len(c.attrs); n++ {
		c.attrs[n] = h.prefixed(c.attrs[n])
	}
	return &c
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	c := *h
	c.inner = h.inner.WithGroup(name)
	c.prefix += name + "."
	return &c
}

func (h *Handler) prefixed(a slog.Attr) slog.Attr {
	a.Key = h.prefix + a.Key
	return a
}

// run sends the queued entries, those logged within batchDelay of each other together,
// until stop; then it sends what is left and closes done.
func (s *sink) run() {
	defer close(s.done)
	for {
		var batch []entry
		select {
		case e := <-s.entries:
			batch = append(batch, e)
		case <-s.stop:
			for {
				select {
				case e := <-s.entries:
					batch = append(batch, e)
				default:
					s.send(batch)
					return
				}
			}
		}
		timer := time.NewTimer(batchDelay)
	gather:
		for {
			select {
			case e := <-s.entries:
				batch = append(batch, e)
			case <-timer.C:
				break gather
			}
		}
		s.send(batch)
	}
}

// send posts the entries, as many webhook messages as Discord's limits require.
// discordgo waits out the webhook's rate limit.
func (s *sink) send(batch []entry) {
	embeds := make([]*discordgo.MessageEmbed, len(batch))
	for n, e := range batch {
		embeds[n] = s.embed(e)
	}
	if dropped := s.dropped.Swap(0); dropped > 0 {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title: fmt.Sprintf("[WARNING] %d logs dropped", dropped),
			Color: colorWarn,
		})
	}
	for len(embeds) > 0 {
		n, chars := 0, 0
		for n < len(embeds) && n < maxEmbeds && (n == 0 || chars+embedChars(embeds[n]) <= maxChars) {
			chars += embedChars(embeds[n])
			n++
		}
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		_, err := s.session.WebhookExecute(s.webhookID, s.webhookToken, false, &discordgo.WebhookParams{
			Embeds:          embeds[:n],
			AllowedMentions: &discordgo.MessageAllowedMentions{}, // never ping
		}, discordgo.WithContext(ctx))
		cancel()
		if err != nil {
			s.errLog.Error("send logs to the Discord webhook", "logs", n, "err", err)
		}
		embeds = embeds[n:]
	}
}

// embed shows an entry as "[LEVEL] message", its attributes one per line.
func (s *sink) embed(e entry) *discordgo.MessageEmbed {
	label, color := "LOG", colorLog
	switch {
	case e.level >= slog.LevelError:
		label, color = "ERROR", colorError
	case e.level >= slog.LevelWarn:
		label, color = "WARNING", colorWarn
	case e.level < slog.LevelInfo:
		label = "DEBUG"
	}
	var b strings.Builder
	for _, a := range e.attrs {
		s.writeAttr(&b, "", a)
	}
	embed := &discordgo.MessageEmbed{
		Title:       truncate("["+label+"] "+e.msg, 200),
		Description: truncate(b.String(), 3000),
		Color:       color,
	}
	if !e.time.IsZero() {
		embed.Timestamp = e.time.Format(time.RFC3339)
	}
	return embed
}

func (s *sink) writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		for _, sub := range v.Group() {
			s.writeAttr(b, prefix+a.Key+".", sub)
		}
		return
	}
	if a.Key == "" {
		return
	}
	value := truncate(v.String(), maxValue)
	if a.Key == "guild" {
		if name := s.nameOf(value); name != "" {
			value = name + " (`" + value + "`)"
		}
	}
	fmt.Fprintf(b, "**%s** : %s\n", prefix+a.Key, value)
}

func (s *sink) nameOf(guildID string) string {
	if name := s.guildName.Load(); name != nil {
		return (*name)(guildID)
	}
	return ""
}

func embedChars(e *discordgo.MessageEmbed) int {
	return utf8.RuneCountInString(e.Title) + utf8.RuneCountInString(e.Description)
}

// truncate cuts s to at most n runes, marking the cut with "…".
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
