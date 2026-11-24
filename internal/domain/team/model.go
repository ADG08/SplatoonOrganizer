package team

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// Display selects how the weekly availability message is shown.
type Display string

const (
	// DisplayClassic is the original message: an embed with a text table.
	DisplayClassic Display = "classic"
	// DisplayImage draws the grid as a picture inside a Components V2 container.
	DisplayImage Display = "image"
)

const (
	// MaxTeamsPerGuild keeps the team list within Discord's limits: 4 rows of 5 buttons.
	MaxTeamsPerGuild = 20
	// MaxNameLength is the maximum length of a team's name, in characters.
	MaxNameLength = 32
	// SetupSteps is the number of steps of a team's setup: created, then its channel. The
	// role is optional: it only mentions the players.
	SetupSteps = 2
)

var (
	ErrNotFound     = errors.New("team not found")
	ErrNameEmpty    = errors.New("team name is empty")
	ErrNameTooLong  = errors.New("team name is too long")
	ErrNameTaken    = errors.New("team name already used in this guild")
	ErrTooManyTeams = errors.New("too many teams in this guild")
)

// Team is a group of players of a guild: it has its own settings, weekly message and
// availabilities. Empty strings mean no channel yet / no role ping.
type Team struct {
	ID         int64
	GuildID    string
	Name       string
	ChannelID  string
	RoleID     string
	SlotLayout availability.Layout   // how days are split into slots (half-day by default)
	Display    Display               // how the weekly message is shown (classic by default)
	Language   i18n.Lang             // language of the bot for the team (French by default)
	Schedule   availability.Schedule // when the weekly message is published (Monday 10h by default)
	// Location is the timezone of the team's hours: slots, grid, weeks and publication
	// (availability.DefaultTimezone by default).
	Location *time.Location
	// Active is false while the bot is out of the guild or can't post in the channel:
	// nothing is published, and the settings are kept for when it comes back.
	Active bool
	// MemberTimezones is on for international teams: each member is asked their
	// timezone and sees the hourly slots in it. Off, everyone uses Location.
	MemberTimezones bool
	// FullAt is the players from which a slot is shown full (green) in the grid image;
	// 0 rates the slots against the week's best one.
	FullAt int
}

// MemberTime reports whether members see the slots in their own timezone: hourly slots
// (half-day ones have no hours) of a team that turned MemberTimezones on.
func (t *Team) MemberTime() bool {
	return t.SlotLayout.IsHourly() && t.MemberTimezones
}

// Messages returns the texts in the team's language.
func (t *Team) Messages() *i18n.Messages {
	return i18n.For(t.Language)
}

// StepsDone returns how many setup steps are done, the team existing being the first one.
func (t *Team) StepsDone() int {
	done := 1
	if t.ChannelID != "" {
		done++
	}
	return done
}

// IsComplete reports whether every setup step is done: the team can be published.
func (t *Team) IsComplete() bool {
	return t.StepsDone() == SetupSteps
}

// NormalizeName trims and validates a team's name.
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNameEmpty
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return "", ErrNameTooLong
	}
	return name, nil
}
