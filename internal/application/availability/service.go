package availability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// Service implements availability use cases.
type Service struct {
	repo Repository
	loc  *time.Location
}

// NewService creates a new availability service.
func NewService(repo Repository, loc *time.Location) *Service {
	if loc == nil {
		loc = time.UTC
	}
	return &Service{
		repo: repo,
		loc:  loc,
	}
}

// Location returns the timezone used by the service.
func (s *Service) Location() *time.Location {
	return s.loc
}

// CurrentWeek returns the current week key.
func (s *Service) CurrentWeek() availability.WeekKey {
	return availability.CurrentWeekKey(s.loc)
}

// ToggleAvailability toggles a user's availability for a given day/slot.
func (s *Service) ToggleAvailability(ctx context.Context, userID string, dayIndex, slotIndex int) (availability.WeekKey, error) {
	week := s.CurrentWeek()

	current, err := s.repo.GetUserAvailability(ctx, week, userID)
	if err != nil {
		return "", err
	}

	daySlots := current[dayIndex]
	available := !daySlots[slotIndex]

	if err := s.repo.SetAvailability(ctx, week, userID, dayIndex, slotIndex, available); err != nil {
		return "", err
	}

	return week, nil
}

// SetDayAvailability sets a user's availability for an entire day.
// Selecting at least one slot clears any "unavailable for the week" status.
func (s *Service) SetDayAvailability(ctx context.Context, userID string, dayIndex int, slots [availability.SlotsPerDay]bool) (availability.WeekKey, error) {
	week := s.CurrentWeek()
	anyAvailable := false
	for slotIndex := 0; slotIndex < availability.SlotsPerDay; slotIndex++ {
		if err := s.repo.SetAvailability(ctx, week, userID, dayIndex, slotIndex, slots[slotIndex]); err != nil {
			return "", err
		}
		if slots[slotIndex] {
			anyAvailable = true
		}
	}
	if anyAvailable {
		if err := s.repo.DeleteWeekUnavailable(ctx, week, userID); err != nil {
			return "", err
		}
	}
	return week, nil
}

// ToggleWeekUnavailable toggles a user's "unavailable for the whole week" status.
// Marking unavailable also clears all the user's slot selections for the week.
// It returns the new state (true = now marked unavailable).
func (s *Service) ToggleWeekUnavailable(ctx context.Context, userID string) (availability.WeekKey, bool, error) {
	week := s.CurrentWeek()

	unavailable, err := s.repo.IsUserWeekUnavailable(ctx, week, userID)
	if err != nil {
		return "", false, err
	}

	if unavailable {
		if err := s.repo.DeleteWeekUnavailable(ctx, week, userID); err != nil {
			return "", false, err
		}
		return week, false, nil
	}

	if err := s.repo.DeleteAllUserAvailability(ctx, week, userID); err != nil {
		return "", false, err
	}
	if err := s.repo.SetWeekUnavailable(ctx, week, userID); err != nil {
		return "", false, err
	}
	return week, true, nil
}

// IsUserWeekUnavailable reports whether the user is marked unavailable for the current week.
func (s *Service) IsUserWeekUnavailable(ctx context.Context, userID string) (bool, error) {
	return s.repo.IsUserWeekUnavailable(ctx, s.CurrentWeek(), userID)
}

// ClearWeekUnavailable removes a user's "unavailable for the week" status for the current week.
// Opening "Mes dispos" makes the user available again.
func (s *Service) ClearWeekUnavailable(ctx context.Context, userID string) error {
	return s.repo.DeleteWeekUnavailable(ctx, s.CurrentWeek(), userID)
}

// GetWeekUnavailableUsers returns the user IDs marked unavailable for a week.
func (s *Service) GetWeekUnavailableUsers(ctx context.Context, week availability.WeekKey) ([]string, error) {
	return s.repo.GetWeekUnavailableUsers(ctx, week)
}

// GetUserAvailability returns a user's availability for the current week.
func (s *Service) GetUserAvailability(ctx context.Context, userID string) (availability.WeekKey, map[int]map[int]bool, error) {
	week := s.CurrentWeek()
	data, err := s.repo.GetUserAvailability(ctx, week, userID)
	if err != nil {
		return "", nil, err
	}
	return week, data, nil
}

// UpsertSondageMessage stores the sondage message ID for a week.
func (s *Service) UpsertSondageMessage(ctx context.Context, week availability.WeekKey, messageID string) error {
	return s.repo.UpsertSondageMessage(ctx, week, messageID)
}

// GetSondageMessageID returns the sondage message ID for a week.
func (s *Service) GetSondageMessageID(ctx context.Context, week availability.WeekKey) (string, error) {
	return s.repo.GetSondageMessage(ctx, week)
}

// ListSondageMessageIDs returns all known sondage message IDs.
func (s *Service) ListSondageMessageIDs(ctx context.Context) ([]string, error) {
	return s.repo.ListSondageMessageIDs(ctx)
}

// GetAvailabilitySummary returns availability counts per slot for a week.
func (s *Service) GetAvailabilitySummary(ctx context.Context, week availability.WeekKey) ([]availability.SlotCount, error) {
	return s.repo.GetAvailabilityCounts(ctx, week)
}

// GetAvailabilityUsers returns users per slot for a week.
func (s *Service) GetAvailabilityUsers(ctx context.Context, week availability.WeekKey) ([]availability.SlotUsers, error) {
	return s.repo.GetAvailabilityUsers(ctx, week)
}

// FormatTable formats availability counts as a markdown table string.
func (s *Service) FormatTable(counts []availability.SlotCount) string {
	labelsDays := []string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"}
	slotIndices := []int{0, 1}
	labelsSlots := []string{"Après-midi", "Soir"}

	grid := make([][]int, availability.DaysPerWeek)
	for i := range grid {
		grid[i] = make([]int, availability.SlotsPerDay)
	}
	for _, c := range counts {
		if c.DayIndex >= 0 && c.DayIndex < availability.DaysPerWeek && c.SlotIndex >= 0 && c.SlotIndex < availability.SlotsPerDay {
			grid[c.DayIndex][c.SlotIndex] = c.Count
		}
	}

	var b strings.Builder
	b.WriteString("```md\n")
	fmt.Fprintf(&b, "%-10s %-11s %-11s\n", "Jour", labelsSlots[0], labelsSlots[1])

	for day := 0; day < availability.DaysPerWeek; day++ {
		fmt.Fprintf(&b, "%-10s", labelsDays[day])
		for _, slot := range slotIndices {
			fmt.Fprintf(&b, " %-11d", grid[day][slot])
		}
		b.WriteString("\n")
	}

	b.WriteString("```")
	return b.String()
}

// FormatUsersBySlot formats users per slot as a human-readable string.
func (s *Service) FormatUsersBySlot(users []availability.SlotUsers) string {
	labelsDays := []string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"}
	labelsSlots := []string{"Après-midi", "Soir"}

	type key struct {
		day  int
		slot int
	}

	group := make(map[key][]string)
	for _, u := range users {
		k := key{day: u.DayIndex, slot: u.SlotIndex}
		for _, id := range u.UserIDs {
			group[k] = append(group[k], "<@"+id+">")
		}
	}

	var b strings.Builder
	for day := 0; day < availability.DaysPerWeek; day++ {
		parts := make([]string, 0, availability.SlotsPerDay)
		for slot := 0; slot < availability.SlotsPerDay; slot++ {
			k := key{day: day, slot: slot}
			mentions := group[k]
			value := "—"
			if len(mentions) > 0 {
				value = strings.Join(mentions, ", ")
			}
			parts = append(parts, labelsSlots[slot]+" : "+value)
		}
		fmt.Fprintf(&b, "• **%s** · %s\n", labelsDays[day], strings.Join(parts, " · "))
	}
	return b.String()
}

// FormatUnavailableUsers formats the users marked unavailable for the week as a mention list.
// Returns an empty string when nobody is marked unavailable.
func (s *Service) FormatUnavailableUsers(userIDs []string) string {
	if len(userIDs) == 0 {
		return ""
	}
	mentions := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		mentions = append(mentions, "<@"+id+">")
	}
	return "🚫 **Indisponibles cette semaine** : " + strings.Join(mentions, ", ")
}
