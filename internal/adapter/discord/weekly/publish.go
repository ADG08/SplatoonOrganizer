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
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/bwmarrin/discordgo"
)

// Timeout bounds one publication or refresh: rendering the image, then a few Discord calls.
const Timeout = 30 * time.Second

// errNoChannel is returned when publishing for a team that has no channel configured.
var errNoChannel = errors.New("no channel configured")

// teamLocks serializes what is done to each team's weekly message: the scheduler, /help
// and the refreshes after clicks run concurrently, and two publications at once would
// post the message twice, or an edit would hit a message being replaced.
var teamLocks sync.Map // team ID -> *sync.Mutex

// lockTeam locks the team's weekly message and returns the function unlocking it.
func lockTeam(teamID int64) (unlock func()) {
	mu, _ := teamLocks.LoadOrStore(teamID, new(sync.Mutex))
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

// Publish replaces the team's weekly message: it posts a fresh one in the configured
// channel with the team's layout and display, then deletes the previous one.
// Used by the weekly job and when /help changes how the message looks (an embed can't
// always be edited into a Components V2 message, so it is posted again instead).
func Publish(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, t *team.Team) error {
	defer lockTeam(t.ID)()
	return publish(ctx, s, svc, t)
}

// PublishDue publishes the team's weekly message when Due, and reports whether it did.
// Checking and publishing under the same lock keeps two checks from both publishing.
func PublishDue(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, t *team.Team) (bool, error) {
	defer lockTeam(t.ID)()
	if due, err := Due(ctx, svc, t); err != nil || !due {
		return false, err
	}
	return true, publish(ctx, s, svc, t)
}

// publish is Publish, the team being locked.
func publish(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, t *team.Team) error {
	if t.ChannelID == "" {
		return errNoChannel
	}
	week := svc.Week(t.Schedule, t.Location)
	counts, err := svc.AvailabilitySummary(ctx, t.ID, week)
	if err != nil {
		return fmt.Errorf("get availability summary: %w", err)
	}
	m := t.Messages()
	payload, err := BuildPayload(t, svc.BuildWeeklyView(m, t.SlotLayout, t.Location, t.Schedule, week, counts))
	if err != nil {
		return err
	}

	// Unknown, the previous message couldn't be deleted: don't post a second one.
	previous, err := svc.WeeklyMessage(ctx, t.ID)
	if err != nil {
		return fmt.Errorf("get weekly message: %w", err)
	}

	// The new message is posted first: if that fails (a permission missing, such as
	// attaching the image), the previous one stays in place.
	msg, err := s.ChannelMessageSendComplex(t.ChannelID, payload.MessageSend(), discordgo.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("send weekly message: %w", err)
	}
	if err := svc.SetWeeklyMessage(ctx, t.ID, availability.WeeklyMessage{Week: week, ChannelID: t.ChannelID, MessageID: msg.ID}); err != nil {
		// Keep the message the database knows of, and remove the one just posted.
		if err := s.ChannelMessageDelete(t.ChannelID, msg.ID, discordgo.WithContext(ctx)); err != nil {
			slog.ErrorContext(ctx, "delete unsaved weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		}
		return fmt.Errorf("save weekly message: %w", err)
	}

	if previous != nil {
		if err := s.ChannelMessageDelete(cmp.Or(previous.ChannelID, t.ChannelID), previous.MessageID, discordgo.WithContext(ctx)); err != nil {
			slog.ErrorContext(ctx, "delete previous weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		}
	}
	slog.InfoContext(ctx, "weekly message published", "guild", t.GuildID, "team", t.Name, "week", week)
	return nil
}

// Remove deletes the team's weekly message from Discord (best-effort): the team is going away.
func Remove(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, t *team.Team) {
	defer lockTeam(t.ID)()
	msg, err := svc.WeeklyMessage(ctx, t.ID)
	if err != nil {
		slog.ErrorContext(ctx, "get weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		return
	}
	if msg == nil {
		return
	}
	if err := s.ChannelMessageDelete(cmp.Or(msg.ChannelID, t.ChannelID), msg.MessageID, discordgo.WithContext(ctx)); err != nil {
		slog.ErrorContext(ctx, "delete weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
	}
}

// LostAccess reports whether err means the bot can no longer post in the team's
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
func PublishInBackground(ctx context.Context, s *discordgo.Session, svc *appavailability.Service, t *team.Team) {
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), Timeout)
		defer cancel()
		if err := Publish(ctx, s, svc, t); err != nil {
			slog.ErrorContext(ctx, "publish weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		}
	}()
}

// Due reports whether the team's current week has no weekly message yet:
// its publication time passed, possibly while the bot was down.
func Due(ctx context.Context, svc *appavailability.Service, t *team.Team) (bool, error) {
	msg, err := svc.WeeklyMessage(ctx, t.ID)
	if err != nil {
		return false, fmt.Errorf("get weekly message: %w", err)
	}
	return msg == nil || msg.Week != svc.Week(t.Schedule, t.Location), nil
}

// refreshCooldown bounds how often a team's weekly message is edited: a burst of clicks
// is merged into one edit per cooldown (rendering the image costs CPU, and Discord rate
// limits message edits per channel).
const refreshCooldown = time.Second

// Refresher updates the weekly message after availabilities change. The first
// request of a team is applied right away; the ones arriving during the cooldown that
// follows are merged into a single extra update.
type Refresher struct {
	svc   *appavailability.Service
	teams *appteam.Service

	mu    sync.Mutex
	dirty map[int64]bool // teams being refreshed -> changed again since
}

// NewRefresher creates a refresher of the weekly messages.
func NewRefresher(svc *appavailability.Service, teams *appteam.Service) *Refresher {
	return &Refresher{svc: svc, teams: teams, dirty: make(map[int64]bool)}
}

// Request asks for the team's weekly message to be updated.
func (r *Refresher) Request(s *discordgo.Session, teamID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, running := r.dirty[teamID]; running {
		r.dirty[teamID] = true
		return
	}
	r.dirty[teamID] = false
	go r.run(s, teamID)
}

func (r *Refresher) run(s *discordgo.Session, teamID int64) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		r.refresh(ctx, s, teamID)
		cancel()
		time.Sleep(refreshCooldown)

		r.mu.Lock()
		if !r.dirty[teamID] {
			delete(r.dirty, teamID)
			r.mu.Unlock()
			return
		}
		r.dirty[teamID] = false
		r.mu.Unlock()
	}
}

// refresh re-renders the team's weekly message of the current week (best-effort).
func (r *Refresher) refresh(ctx context.Context, s *discordgo.Session, teamID int64) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "panic while refreshing weekly message", "team_id", teamID, "panic", rec)
		}
	}()
	defer lockTeam(teamID)()

	t, err := r.teams.Get(ctx, teamID)
	if errors.Is(err, team.ErrNotFound) {
		return // deleted meanwhile, with its message
	}
	if err != nil {
		slog.ErrorContext(ctx, "get team", "team_id", teamID, "err", err)
		return
	}
	week := r.svc.Week(t.Schedule, t.Location)
	msg, err := r.svc.WeeklyMessage(ctx, teamID)
	if err != nil || msg == nil || msg.Week != week {
		if err != nil {
			slog.ErrorContext(ctx, "get weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		}
		return
	}
	counts, err := r.svc.AvailabilitySummary(ctx, teamID, week)
	if err != nil {
		slog.ErrorContext(ctx, "get availability summary", "guild", t.GuildID, "team", t.Name, "err", err)
		return
	}
	m := t.Messages()
	payload, err := BuildPayload(t, r.svc.BuildWeeklyView(m, t.SlotLayout, t.Location, t.Schedule, week, counts))
	if err != nil {
		slog.ErrorContext(ctx, "build weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
		return
	}
	if _, err := s.ChannelMessageEditComplex(payload.MessageEdit(cmp.Or(msg.ChannelID, t.ChannelID), msg.MessageID), discordgo.WithContext(ctx)); err != nil {
		slog.ErrorContext(ctx, "edit weekly message", "guild", t.GuildID, "team", t.Name, "err", err)
	}
}
