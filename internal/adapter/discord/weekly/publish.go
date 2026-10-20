package weekly

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// Timeout bounds one publication or refresh: rendering the image, then a few Discord calls.
const Timeout = 30 * time.Second

// errNoChannel is returned when publishing for a guild that has no channel configured.
var errNoChannel = errors.New("no channel configured")

// guildLocks serializes what is done to each guild's weekly message: the scheduler, /help
// and the refreshes after clicks run concurrently, and two publications at once would
// post the message twice, or an edit would hit a message being replaced.
var guildLocks sync.Map // guildID -> *sync.Mutex

// lockGuild locks the guild's weekly message and returns the function unlocking it.
func lockGuild(guildID string) (unlock func()) {
	mu, _ := guildLocks.LoadOrStore(guildID, new(sync.Mutex))
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

// Publish replaces the guild's weekly message: it posts a fresh one in the configured
// channel with the guild's layout and display, then deletes the previous one.
// Used by the weekly job and when /help changes how the message looks (an embed can't
// always be edited into a Components V2 message, so it is posted again instead).
func Publish(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig) error {
	defer lockGuild(cfg.GuildID)()
	return publish(ctx, s, svc, cfg)
}

// PublishDue publishes the guild's weekly message when Due, and reports whether it did.
// Checking and publishing under the same lock keeps two checks from both publishing.
func PublishDue(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig) (bool, error) {
	defer lockGuild(cfg.GuildID)()
	if due, err := Due(ctx, svc, cfg); err != nil || !due {
		return false, err
	}
	return true, publish(ctx, s, svc, cfg)
}

// publish is Publish, the guild being locked.
func publish(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig) error {
	if cfg.ChannelID == "" {
		return errNoChannel
	}
	week := svc.Week(cfg.Schedule, cfg.Location)
	counts, err := svc.AvailabilitySummary(ctx, cfg.GuildID, week)
	if err != nil {
		return fmt.Errorf("get availability summary: %w", err)
	}
	m := cfg.Messages()
	payload, err := BuildPayload(cfg, svc.BuildWeeklyView(m, cfg.SlotLayout, cfg.Location, cfg.Schedule, week, counts))
	if err != nil {
		return err
	}

	// Unknown, the previous message couldn't be deleted: don't post a second one.
	previous, err := svc.WeeklyMessage(ctx, cfg.GuildID)
	if err != nil {
		return fmt.Errorf("get weekly message: %w", err)
	}

	// The new message is posted first: if that fails (a permission missing, such as
	// attaching the image), the previous one stays in place.
	msg, err := s.ChannelMessageSendComplex(cfg.ChannelID, payload.MessageSend(), discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("send weekly message: %w", err)
	}
	if err := svc.SetWeeklyMessage(ctx, cfg.GuildID, availability.WeeklyMessage{Week: week, ChannelID: cfg.ChannelID, MessageID: msg.ID}); err != nil {
		// Keep the message the database knows of, and remove the one just posted.
		if err := s.ChannelMessageDelete(cfg.ChannelID, msg.ID, discordgo.WithContext(ctx)); err != nil {
			slog.ErrorContext(ctx, "delete unsaved weekly message", "guild", cfg.GuildID, "err", err)
		}
		return fmt.Errorf("save weekly message: %w", err)
	}

	if previous != nil {
		if err := s.ChannelMessageDelete(cmp.Or(previous.ChannelID, cfg.ChannelID), previous.MessageID, discordgo.WithContext(ctx)); err != nil {
			slog.ErrorContext(ctx, "delete previous weekly message", "guild", cfg.GuildID, "err", err)
		}
	}
	slog.InfoContext(ctx, "weekly message published", "guild", cfg.GuildID, "week", week)
	return nil
}

// LostAccess reports whether err means the bot can no longer post in the guild's
// channel: the channel or the guild is gone, or the bot lost its permissions there.
func LostAccess(err error) bool {
	var rest *discordgo.RESTError
	if !errors.As(err, &rest) || rest.Message == nil {
		return false
	}
	switch rest.Message.Code {
	case discordgo.ErrCodeUnknownChannel, discordgo.ErrCodeUnknownGuild,
		discordgo.ErrCodeMissingAccess, discordgo.ErrCodeMissingPermissions:
		return true
	}
	return false
}

// PublishInBackground publishes for an interaction that is answered meanwhile: the
// publication keeps ctx's values but not its cancellation, which comes with the answer.
func PublishInBackground(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), Timeout)
		defer cancel()
		if err := Publish(ctx, s, svc, cfg); err != nil {
			slog.ErrorContext(ctx, "publish weekly message", "guild", cfg.GuildID, "err", err)
		}
	}()
}

// Due reports whether the guild's current week has no weekly message yet:
// its publication time passed, possibly while the bot was down.
func Due(ctx context.Context, svc *appavailability.Service, cfg *guildconfig.GuildConfig) (bool, error) {
	msg, err := svc.WeeklyMessage(ctx, cfg.GuildID)
	if err != nil {
		return false, fmt.Errorf("get weekly message: %w", err)
	}
	return msg == nil || msg.Week != svc.Week(cfg.Schedule, cfg.Location), nil
}

// refreshCooldown bounds how often a guild's weekly message is edited: a burst of clicks
// is merged into one edit per cooldown (rendering the image costs CPU, and Discord rate
// limits message edits per channel).
const refreshCooldown = time.Second

// Refresher updates the weekly message after availabilities change. The first
// request of a guild is applied right away; the ones arriving during the cooldown that
// follows are merged into a single extra update.
type Refresher struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service

	mu    sync.Mutex
	dirty map[string]bool // guilds being refreshed -> changed again since
}

// NewRefresher creates a refresher of the weekly messages.
func NewRefresher(svc *appavailability.Service, guildCfg *appguildconfig.Service) *Refresher {
	return &Refresher{svc: svc, guildCfg: guildCfg, dirty: make(map[string]bool)}
}

// Request asks for the guild's weekly message to be updated.
func (r *Refresher) Request(s *discordgo.Session, guildID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, running := r.dirty[guildID]; running {
		r.dirty[guildID] = true
		return
	}
	r.dirty[guildID] = false
	go r.run(s, guildID)
}

func (r *Refresher) run(s *discordgo.Session, guildID string) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		r.refresh(ctx, s, guildID)
		cancel()
		time.Sleep(refreshCooldown)

		r.mu.Lock()
		if !r.dirty[guildID] {
			delete(r.dirty, guildID)
			r.mu.Unlock()
			return
		}
		r.dirty[guildID] = false
		r.mu.Unlock()
	}
}

// refresh re-renders the guild's weekly message of the current week (best-effort).
func (r *Refresher) refresh(ctx context.Context, s *discordgo.Session, guildID string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "panic while refreshing weekly message", "guild", guildID, "panic", rec)
		}
	}()
	defer lockGuild(guildID)()

	cfg, err := r.guildCfg.Get(ctx, guildID)
	if err != nil {
		slog.ErrorContext(ctx, "get guild config", "guild", guildID, "err", err)
		return
	}
	week := r.svc.Week(cfg.Schedule, cfg.Location)
	msg, err := r.svc.WeeklyMessage(ctx, guildID)
	if err != nil || msg == nil || msg.Week != week {
		if err != nil {
			slog.ErrorContext(ctx, "get weekly message", "guild", guildID, "err", err)
		}
		return
	}
	counts, err := r.svc.AvailabilitySummary(ctx, guildID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability summary", "guild", guildID, "err", err)
		return
	}
	m := cfg.Messages()
	payload, err := BuildPayload(cfg, r.svc.BuildWeeklyView(m, cfg.SlotLayout, cfg.Location, cfg.Schedule, week, counts))
	if err != nil {
		slog.ErrorContext(ctx, "build weekly message", "guild", guildID, "err", err)
		return
	}
	if _, err := s.ChannelMessageEditComplex(payload.MessageEdit(cmp.Or(msg.ChannelID, cfg.ChannelID), msg.MessageID), discordgo.WithContext(ctx)); err != nil {
		slog.ErrorContext(ctx, "edit weekly message", "guild", guildID, "err", err)
	}
}
