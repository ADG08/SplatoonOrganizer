package weekly

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

// ErrNoChannel is returned when publishing for a guild that has no channel configured.
var ErrNoChannel = errors.New("no channel configured")

// Publish replaces the guild's weekly message: it deletes the previous one,
// then posts a fresh one in the configured channel with the guild's layout and display.
// Used by the weekly job and when /help changes how the message looks (an embed can't
// always be edited into a Components V2 message, so it is posted again instead).
func Publish(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, cfg *guildconfig.GuildConfig) error {
	if cfg.ChannelID == "" {
		return ErrNoChannel
	}
	week := svc.Week(cfg.Schedule)
	counts, err := svc.AvailabilitySummary(ctx, cfg.GuildID, week)
	if err != nil {
		return fmt.Errorf("get availability summary: %w", err)
	}
	m := cfg.Messages()
	payload, err := BuildPayload(m, svc.BuildWeeklyView(m, cfg.SlotLayout, week, counts), cfg.RoleID, cfg.Display)
	if err != nil {
		return err
	}

	previous, err := svc.WeeklyMessage(ctx, cfg.GuildID)
	if err != nil {
		log.Printf("guild %s: get weekly message: %v", cfg.GuildID, err)
	} else if previous != nil {
		if err := s.ChannelMessageDelete(cmp.Or(previous.ChannelID, cfg.ChannelID), previous.MessageID); err != nil {
			log.Printf("guild %s: delete previous weekly message: %v", cfg.GuildID, err)
		}
	}

	msg, err := s.ChannelMessageSendComplex(cfg.ChannelID, payload.MessageSend())
	if err != nil {
		return fmt.Errorf("send weekly message: %w", err)
	}
	return svc.SetWeeklyMessage(ctx, cfg.GuildID, availability.WeeklyMessage{Week: week, ChannelID: cfg.ChannelID, MessageID: msg.ID})
}

// Due reports whether the guild's current week has no weekly message yet:
// its publication time passed, possibly while the bot was down.
func Due(ctx context.Context, svc *appavailability.Service, cfg *guildconfig.GuildConfig) (bool, error) {
	msg, err := svc.WeeklyMessage(ctx, cfg.GuildID)
	if err != nil {
		return false, fmt.Errorf("get weekly message: %w", err)
	}
	return msg == nil || msg.Week != svc.Week(cfg.Schedule), nil
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
		r.refresh(context.Background(), s, guildID)
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
			log.Printf("guild %s: panic while refreshing weekly message: %v", guildID, rec)
		}
	}()

	cfg, err := r.guildCfg.Get(ctx, guildID)
	if err != nil {
		log.Printf("guild %s: get guild config: %v", guildID, err)
		return
	}
	week := r.svc.Week(cfg.Schedule)
	msg, err := r.svc.WeeklyMessage(ctx, guildID)
	if err != nil || msg == nil || msg.Week != week {
		if err != nil {
			log.Printf("guild %s: get weekly message: %v", guildID, err)
		}
		return
	}
	counts, err := r.svc.AvailabilitySummary(ctx, guildID, week)
	if err != nil {
		log.Printf("guild %s: get availability summary: %v", guildID, err)
		return
	}
	m := cfg.Messages()
	payload, err := BuildPayload(m, r.svc.BuildWeeklyView(m, cfg.SlotLayout, week, counts), cfg.RoleID, cfg.Display)
	if err != nil {
		log.Printf("guild %s: build weekly message: %v", guildID, err)
		return
	}
	if _, err := s.ChannelMessageEditComplex(payload.MessageEdit(cmp.Or(msg.ChannelID, cfg.ChannelID), msg.MessageID)); err != nil {
		log.Printf("guild %s: edit weekly message: %v", guildID, err)
	}
}
