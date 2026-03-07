package postgres

import (
	"context"
	"fmt"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AvailabilityRepository implements availability.Repository using Postgres.
type AvailabilityRepository struct {
	queries *db.Queries
}

// NewAvailabilityRepository creates a new Postgres availability repository.
func NewAvailabilityRepository(pool *pgxpool.Pool) *AvailabilityRepository {
	return &AvailabilityRepository{
		queries: db.New(pool),
	}
}

// Ensure AvailabilityRepository implements the interface.
var _ appavailability.Repository = (*AvailabilityRepository)(nil)

func (r *AvailabilityRepository) UpsertSondageMessage(ctx context.Context, week availability.WeekKey, messageID string) error {
	if err := r.queries.UpsertSondageMessage(ctx, db.UpsertSondageMessageParams{
		Week:      string(week),
		MessageID: messageID,
	}); err != nil {
		return fmt.Errorf("upsert sondage message: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) GetSondageMessage(ctx context.Context, week availability.WeekKey) (string, error) {
	row, err := r.queries.GetSondageMessageByWeek(ctx, string(week))
	if err != nil {
		return "", fmt.Errorf("get sondage message: %w", err)
	}
	return row.MessageID, nil
}

func (r *AvailabilityRepository) ListSondageMessageIDs(ctx context.Context) ([]string, error) {
	ids, err := r.queries.ListSondageMessageIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sondage message ids: %w", err)
	}
	return ids, nil
}

func (r *AvailabilityRepository) GetUserAvailability(ctx context.Context, week availability.WeekKey, userID string) (map[int]map[int]bool, error) {
	rows, err := r.queries.GetUserAvailability(ctx, db.GetUserAvailabilityParams{
		Week:   string(week),
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("get user availability: %w", err)
	}

	result := make(map[int]map[int]bool)
	for _, row := range rows {
		day := int(row.DayIndex)
		slot := int(row.SlotIndex)
		if _, ok := result[day]; !ok {
			result[day] = make(map[int]bool)
		}
		result[day][slot] = true
	}

	return result, nil
}

func (r *AvailabilityRepository) SetAvailability(ctx context.Context, week availability.WeekKey, userID string, dayIndex, slotIndex int, available bool) error {
	if available {
		if err := r.queries.InsertAvailability(ctx, db.InsertAvailabilityParams{
			UserID:    userID,
			DayIndex:  int16(dayIndex),
			SlotIndex: int16(slotIndex),
			Week:      string(week),
		}); err != nil {
			return fmt.Errorf("insert availability: %w", err)
		}
		return nil
	}

	if err := r.queries.DeleteAvailability(ctx, db.DeleteAvailabilityParams{
		UserID:    userID,
		DayIndex:  int16(dayIndex),
		SlotIndex: int16(slotIndex),
		Week:      string(week),
	}); err != nil {
		return fmt.Errorf("delete availability: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) GetAvailabilityCounts(ctx context.Context, week availability.WeekKey) ([]availability.SlotCount, error) {
	rows, err := r.queries.GetAvailabilityCounts(ctx, string(week))
	if err != nil {
		return nil, fmt.Errorf("get availability counts: %w", err)
	}

	counts := make([]availability.SlotCount, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, availability.SlotCount{
			DayIndex:  int(row.DayIndex),
			SlotIndex: int(row.SlotIndex),
			Count:     int(row.Count),
		})
	}
	return counts, nil
}

func (r *AvailabilityRepository) GetAvailabilityUsers(ctx context.Context, week availability.WeekKey) ([]availability.SlotUsers, error) {
	rows, err := r.queries.GetAvailabilityUsers(ctx, string(week))
	if err != nil {
		return nil, fmt.Errorf("get availability users: %w", err)
	}

	groups := make(map[[2]int][]string)
	for _, row := range rows {
		key := [2]int{int(row.DayIndex), int(row.SlotIndex)}
		groups[key] = append(groups[key], row.UserID)
	}

	result := make([]availability.SlotUsers, 0, len(groups))
	for key, users := range groups {
		result = append(result, availability.SlotUsers{
			DayIndex:  key[0],
			SlotIndex: key[1],
			UserIDs:   users,
		})
	}

	return result, nil
}
