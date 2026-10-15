package availability

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// Repository is the outbound port for availability persistence. Everything but the
// members' timezones is scoped to a guild.
type Repository interface {
	// WeeklyMessage returns the guild's latest weekly message, or nil when there is none.
	WeeklyMessage(ctx context.Context, guildID string) (*availability.WeeklyMessage, error)
	SetWeeklyMessage(ctx context.Context, guildID string, msg availability.WeeklyMessage) error

	UserAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string) (map[int]map[int]bool, error)
	// SetDayAvailability replaces the member's slots of a day; selecting at least one slot
	// also clears their "unavailable for the week" status.
	SetDayAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string, dayIndex int, slots []int) error
	DeleteAllUserAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string) error
	DeleteWeekAvailability(ctx context.Context, guildID string, week availability.WeekKey) error

	AvailabilityCounts(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotCount, error)
	AvailabilityUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotUsers, error)

	SetWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) error
	DeleteWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) error
	IsUserWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) (bool, error)
	WeekUnavailableUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]string, error)

	// UserTimezone returns the IANA name a member picked, or "" when they never did.
	UserTimezone(ctx context.Context, userID string) (string, error)
	SetUserTimezone(ctx context.Context, userID, timezone string) error
}
