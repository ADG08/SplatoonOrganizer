package availability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
)

// Service implements availability use cases. Except members' timezones, everything is
// scoped to a guild.
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

// SetDayAvailability sets a member's availability for an entire day, in one round trip.
// slots has one entry per slot of the guild's layout.
// Selecting at least one slot clears any "unavailable for the week" status.
func (s *Service) SetDayAvailability(ctx context.Context, guildID, userID string, dayIndex int, slots []bool) (availability.WeekKey, error) {
	week := s.CurrentWeek()
	var selected []int
	for slot, ok := range slots {
		if ok {
			selected = append(selected, slot)
		}
	}
	return week, s.repo.SetDayAvailability(ctx, guildID, week, userID, dayIndex, selected)
}

// ToggleWeekUnavailable toggles a member's "unavailable for the whole week" status.
// Marking unavailable also clears all the member's slot selections for the week.
// It returns the new state (true = now marked unavailable).
func (s *Service) ToggleWeekUnavailable(ctx context.Context, guildID, userID string) (bool, error) {
	week := s.CurrentWeek()

	unavailable, err := s.repo.IsUserWeekUnavailable(ctx, guildID, week, userID)
	if err != nil {
		return false, err
	}

	if unavailable {
		return false, s.repo.DeleteWeekUnavailable(ctx, guildID, week, userID)
	}

	if err := s.repo.DeleteAllUserAvailability(ctx, guildID, week, userID); err != nil {
		return false, err
	}
	return true, s.repo.SetWeekUnavailable(ctx, guildID, week, userID)
}

// ClearWeekUnavailable removes a member's "unavailable for the week" status for the current week.
// Opening "Mes dispos" makes the member available again.
func (s *Service) ClearWeekUnavailable(ctx context.Context, guildID, userID string) error {
	return s.repo.DeleteWeekUnavailable(ctx, guildID, s.CurrentWeek(), userID)
}

// ResetWeek deletes every slot selection of the guild's week. Used when the slot layout
// changes, since slot indexes of the old layout would be misread with the new one.
func (s *Service) ResetWeek(ctx context.Context, guildID string, week availability.WeekKey) error {
	return s.repo.DeleteWeekAvailability(ctx, guildID, week)
}

// GetWeekUnavailableUsers returns the members marked unavailable for a week.
func (s *Service) GetWeekUnavailableUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]string, error) {
	return s.repo.GetWeekUnavailableUsers(ctx, guildID, week)
}

// GetUserAvailability returns a member's availability for the current week.
func (s *Service) GetUserAvailability(ctx context.Context, guildID, userID string) (availability.WeekKey, map[int]map[int]bool, error) {
	week := s.CurrentWeek()
	data, err := s.repo.GetUserAvailability(ctx, guildID, week, userID)
	if err != nil {
		return "", nil, err
	}
	return week, data, nil
}

// WeeklyMessage returns the guild's latest weekly message, or nil when there is none.
func (s *Service) WeeklyMessage(ctx context.Context, guildID string) (*availability.WeeklyMessage, error) {
	return s.repo.GetWeeklyMessage(ctx, guildID)
}

// SetWeeklyMessage remembers the guild's latest weekly message.
func (s *Service) SetWeeklyMessage(ctx context.Context, guildID string, msg availability.WeeklyMessage) error {
	return s.repo.SetWeeklyMessage(ctx, guildID, msg)
}

// GetAvailabilitySummary returns availability counts per slot for a week.
func (s *Service) GetAvailabilitySummary(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotCount, error) {
	return s.repo.GetAvailabilityCounts(ctx, guildID, week)
}

// GetAvailabilityUsers returns members per slot for a week.
func (s *Service) GetAvailabilityUsers(ctx context.Context, guildID string, week availability.WeekKey) ([]availability.SlotUsers, error) {
	return s.repo.GetAvailabilityUsers(ctx, guildID, week)
}

// FormatUsersBySlot formats users per slot as a human-readable string.
func (s *Service) FormatUsersBySlot(layout availability.Layout, week availability.WeekKey, users []availability.SlotUsers) string {
	if layout.IsHourly() {
		return s.formatUsersByHour(layout, week, users)
	}

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
		parts := make([]string, 0, layout.Len())
		for slot := 0; slot < layout.Len(); slot++ {
			k := key{day: day, slot: slot}
			mentions := group[k]
			value := "—"
			if len(mentions) > 0 {
				value = strings.Join(mentions, ", ")
			}
			parts = append(parts, layout.Label(slot)+" : "+value)
		}
		fmt.Fprintf(&b, "• **%s** · %s\n", dayLabels[day], strings.Join(parts, " · "))
	}
	return b.String()
}

// formatUsersByHour lists, per day, each user with their ranges of consecutive hours.
func (s *Service) formatUsersByHour(layout availability.Layout, week availability.WeekKey, users []availability.SlotUsers) string {
	byDay := usersByDay(layout, users)

	var b strings.Builder
	b.WriteString("*🕒 Heures affichées dans ton fuseau horaire*\n")
	for day := 0; day < availability.DaysPerWeek; day++ {
		if len(byDay[day]) == 0 {
			fmt.Fprintf(&b, "\n**%s** · —\n", dayLabels[day])
			continue
		}
		fmt.Fprintf(&b, "\n**%s**\n", dayLabels[day])
		for _, line := range s.dayUserLines(layout, week, day, byDay[day]) {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

// FormatDayUsers lists who is available on one day, one line per user with their ranges.
func (s *Service) FormatDayUsers(layout availability.Layout, week availability.WeekKey, users []availability.SlotUsers, dayIndex int) string {
	byUser := usersByDay(layout, users)[dayIndex]
	if len(byUser) == 0 {
		return "Personne n'est dispo ce jour-là pour l'instant."
	}

	var b strings.Builder
	if layout.IsHourly() {
		b.WriteString("-# 🕒 Heures affichées dans ton fuseau horaire\n")
	}
	b.WriteString(strings.Join(s.dayUserLines(layout, week, dayIndex, byUser), "\n"))
	return b.String()
}

// usersByDay indexes availabilities as byDay[day][userID][slot] = available.
func usersByDay(layout availability.Layout, users []availability.SlotUsers) []map[string][]bool {
	byDay := make([]map[string][]bool, availability.DaysPerWeek)
	for _, u := range users {
		if u.DayIndex < 0 || u.DayIndex >= availability.DaysPerWeek || !layout.Valid(u.SlotIndex) {
			continue
		}
		if byDay[u.DayIndex] == nil {
			byDay[u.DayIndex] = make(map[string][]bool)
		}
		for _, id := range u.UserIDs {
			if byDay[u.DayIndex][id] == nil {
				byDay[u.DayIndex][id] = make([]bool, layout.Len())
			}
			byDay[u.DayIndex][id][u.SlotIndex] = true
		}
	}
	return byDay
}

// dayUserLines renders one line per user of a day, earliest availability first.
// Consecutive hours are merged into one range: 14h to minuit shows as "14:00 – 00:00".
func (s *Service) dayUserLines(layout availability.Layout, week availability.WeekKey, day int, byUser map[string][]bool) []string {
	firstSlot := func(slots []bool) int {
		for slot, ok := range slots {
			if ok {
				return slot
			}
		}
		return len(slots)
	}
	ids := make([]string, 0, len(byUser))
	for id := range byUser {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		fi, fj := firstSlot(byUser[ids[i]]), firstSlot(byUser[ids[j]])
		if fi != fj {
			return fi < fj
		}
		return ids[i] < ids[j]
	})

	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		slots := byUser[id]
		var ranges []string
		for slot := 0; slot < len(slots); slot++ {
			if !slots[slot] {
				continue
			}
			if !layout.IsHourly() {
				ranges = append(ranges, layout.Label(slot))
				continue
			}
			end := slot + 1
			for end < len(slots) && slots[end] {
				end++
			}
			ranges = append(ranges, s.slotTime(layout, week, day, slot)+" – "+s.slotTime(layout, week, day, end))
			slot = end
		}
		lines = append(lines, fmt.Sprintf("<@%s> : %s", id, strings.Join(ranges, ", ")))
	}
	return lines
}

// slotTime renders the start of a slot (or the end of the previous one) as a Discord
// timestamp, which each member sees in their own timezone.
func (s *Service) slotTime(layout availability.Layout, week availability.WeekKey, dayIndex, slot int) string {
	t, err := layout.SlotStart(week, dayIndex, slot, s.loc)
	if err != nil {
		return fmt.Sprintf("%dh", (layout.StartHour+slot)%24)
	}
	return fmt.Sprintf("<t:%d:t>", t.Unix())
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

var (
	dayLabels      = []string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"}
	shortDayLabels = []string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"}
)

// DayLabel returns the full French name of a day (0 = Lundi).
func DayLabel(dayIndex int) string {
	return dayLabels[dayIndex]
}
