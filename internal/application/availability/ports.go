package availability

import (
	"context"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// Repository is the outbound port for availability persistence.
type Repository interface {
	UpsertSondageMessage(ctx context.Context, week availability.WeekKey, messageID string) error
	GetSondageMessage(ctx context.Context, week availability.WeekKey) (string, error)
	ListSondageMessageIDs(ctx context.Context) ([]string, error)

	GetUserAvailability(ctx context.Context, week availability.WeekKey, userID string) (map[int]map[int]bool, error)
	SetAvailability(ctx context.Context, week availability.WeekKey, userID string, dayIndex, slotIndex int, available bool) error

	GetAvailabilityCounts(ctx context.Context, week availability.WeekKey) ([]availability.SlotCount, error)
	GetAvailabilityUsers(ctx context.Context, week availability.WeekKey) ([]availability.SlotUsers, error)
}
