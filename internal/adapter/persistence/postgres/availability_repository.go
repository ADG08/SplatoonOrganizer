package postgres

import (
	"context"
	"errors"
	"fmt"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/db"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/jackc/pgx/v5"
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

func (r *AvailabilityRepository) WeeklyMessage(ctx context.Context, guildID string) (*availability.WeeklyMessage, error) {
	row, err := r.queries.GetWeeklyMessage(ctx, guildID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get weekly message: %w", err)
	}
	return &availability.WeeklyMessage{Week: availability.WeekKey(row.Week), ChannelID: row.ChannelID, MessageID: row.MessageID}, nil
}

func (r *AvailabilityRepository) SetWeeklyMessage(ctx context.Context, guildID string, msg availability.WeeklyMessage) error {
	if err := r.queries.UpsertWeeklyMessage(ctx, db.UpsertWeeklyMessageParams{
		GuildID:   guildID,
		Week:      string(msg.Week),
		ChannelID: msg.ChannelID,
		MessageID: msg.MessageID,
	}); err != nil {
		return fmt.Errorf("upsert weekly message: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) UserAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string) (map[int]map[int]bool, error) {
	rows, err := r.queries.GetUserAvailability(ctx, db.GetUserAvailabilityParams{
		GuildID: guildID,
		Week:    string(week),
		UserID:  userID,
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

func (r *AvailabilityRepository) SetDayAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string, dayIndex int, slots []int) error {
	selected := make([]int16, len(slots))
	for i, slot := range slots {
		selected[i] = int16(slot)
	}
	if err := r.queries.SetDayAvailability(ctx, db.SetDayAvailabilityParams{
		GuildID:  guildID,
		Week:     string(week),
		UserID:   userID,
		DayIndex: int16(dayIndex),
		Slots:    selected,
	}); err != nil {
		return fmt.Errorf("set day availability: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) DeleteAllUserAvailability(ctx context.Context, guildID string, week availability.WeekKey, userID string) error {
	if err := r.queries.DeleteAllUserAvailability(ctx, db.DeleteAllUserAvailabilityParams{
		GuildID: guildID,
		Week:    string(week),
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("delete all user availability: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) DeleteWeekAvailability(ctx context.Context, guildID string, week availability.WeekKey) error {
	if err := r.queries.DeleteWeekAvailability(ctx, db.DeleteWeekAvailabilityParams{
		GuildID: guildID,
		Week:    string(week),
	}); err != nil {
		return fmt.Errorf("delete week availability: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) SetWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) error {
	if err := r.queries.SetWeekUnavailable(ctx, db.SetWeekUnavailableParams{
		GuildID: guildID,
		Week:    string(week),
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("set week unavailable: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) DeleteWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) error {
	if err := r.queries.DeleteWeekUnavailable(ctx, db.DeleteWeekUnavailableParams{
		GuildID: guildID,
		Week:    string(week),
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("delete week unavailable: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) IsUserWeekUnavailable(ctx context.Context, guildID string, week availability.WeekKey, userID string) (bool, error) {
	unavailable, err := r.queries.IsUserWeekUnavailable(ctx, db.IsUserWeekUnavailableParams{
		GuildID: guildID,
		Week:    string(week),
		UserID:  userID,
	})
	if err != nil {
		return false, fmt.Errorf("is user week unavailable: %w", err)
	}
	return unavailable, nil
}

func (r *AvailabilityRepository) WeekUnavailableUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]string, error) {
	users, err := r.queries.GetWeekUnavailableUsers(ctx, db.GetWeekUnavailableUsersParams{
		GuildID: guildID,
		Week:    string(week),
	})
	if err != nil {
		return nil, fmt.Errorf("get week unavailable users: %w", err)
	}
	return users, nil
}

func (r *AvailabilityRepository) UserTimezone(ctx context.Context, userID string) (string, error) {
	zone, err := r.queries.GetUserTimezone(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get user timezone: %w", err)
	}
	return zone, nil
}

func (r *AvailabilityRepository) SetUserTimezone(ctx context.Context, userID, timezone string) error {
	if err := r.queries.SetUserTimezone(ctx, db.SetUserTimezoneParams{UserID: userID, Timezone: timezone}); err != nil {
		return fmt.Errorf("set user timezone: %w", err)
	}
	return nil
}

func (r *AvailabilityRepository) AvailabilityCounts(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotCount, error) {
	rows, err := r.queries.GetAvailabilityCounts(ctx, db.GetAvailabilityCountsParams{
		GuildID: guildID,
		Week:    string(week),
	})
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

func (r *AvailabilityRepository) AvailabilityUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotUsers, error) {
	rows, err := r.queries.GetAvailabilityUsers(ctx, db.GetAvailabilityUsersParams{
		GuildID: guildID,
		Week:    string(week),
	})
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
