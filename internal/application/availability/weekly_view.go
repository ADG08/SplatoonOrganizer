package availability

import (
	"fmt"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
)

// gridMaxColumns keeps the code block narrow enough for phones; wider layouts are split.
const gridMaxColumns = 12

// WeeklyView is the content of the weekly availability message. The Discord adapter
// either shows Table as text (classic display) or draws Counts as an image.
type WeeklyView struct {
	Title  string // e.g. "Dispos de la semaine du 12/10"
	Legend string // timezone hint for hourly layouts, empty otherwise
	Table  string // code block table (the classic display)

	Counts  [][]int  // Counts[day][slot]
	Closed  [][]bool // Closed[day][slot]: starts before the week's publication, can't be picked
	Columns []string // slot headers, e.g. "13h" or "Après-midi"
	Rows    []string // day headers, e.g. "Lun"
	Peak    int      // highest headcount of the week
}

// BuildWeeklyView renders availability counts for the weekly message.
// Discord timestamps don't render inside code blocks or images, so the grid uses the
// reference timezone and only the legend is in each member's timezone.
func (s *Service) BuildWeeklyView(m *i18n.Messages, layout availability.Layout, ref *time.Location, schedule availability.Schedule, week availability.WeekKey, counts []availability.SlotCount) WeeklyView {
	grid := make([][]int, availability.DaysPerWeek)
	for i := range grid {
		grid[i] = make([]int, layout.Len())
	}
	for _, c := range counts {
		if c.DayIndex >= 0 && c.DayIndex < availability.DaysPerWeek && layout.Valid(c.SlotIndex) {
			grid[c.DayIndex][c.SlotIndex] = c.Count
		}
	}

	view := WeeklyView{
		Title:  m.ClassicTitle,
		Counts: grid,
		Closed: availability.ClosedSlots(layout, schedule, week, ref),
		Rows:   append([]string(nil), m.ShortDays[:]...),
	}
	if monday, err := week.Monday(ref); err == nil {
		view.Title = m.WeekTitle(monday)
	}
	for day := range grid {
		for slot, count := range grid[day] {
			view.Peak = max(view.Peak, count)
			if day == 0 {
				view.Columns = append(view.Columns, columnLabel(m, layout, slot))
			}
		}
	}

	if layout.IsHourly() {
		view.Legend = fmt.Sprintf(m.ServerTimeLegend, m.TimezoneName(ref), m.HourLabel(layout.StartHour), s.slotTime(m, layout, ref, week, 0, 0)) + view.closedLegend(m)
		hours := make([]int, layout.Len())
		for slot := range hours {
			hours[slot] = layout.StartHour + slot
		}
		view.Table = hourlyTable(hours, view.Rows, grid, view.Closed)
	} else {
		view.Table = halfDayTable(m, view.Rows, grid)
	}
	return view
}

// closedLegend explains the crossed-out cells, when the week has some.
func (v WeeklyView) closedLegend(m *i18n.Messages) string {
	for _, slots := range v.Closed {
		for _, closed := range slots {
			if closed {
				return " · " + m.ClosedLegend
			}
		}
	}
	return ""
}

func columnLabel(m *i18n.Messages, layout availability.Layout, slot int) string {
	if layout.IsHourly() {
		return m.HourLabel(layout.StartHour + slot)
	}
	return m.HalfDay[slot]
}

// hourlyTable renders one column per hour, labelled with the hour of the day. Closed
// slots are crossed out.
func hourlyTable(hours []int, rows []string, grid [][]int, closed [][]bool) string {
	var b strings.Builder
	b.WriteString("```\n")
	for from := 0; from < len(hours); from += gridMaxColumns {
		to := min(from+gridMaxColumns, len(hours))
		if from > 0 {
			b.WriteString("\n")
		}
		b.WriteString("   ")
		for slot := from; slot < to; slot++ {
			fmt.Fprintf(&b, " %2d", hours[slot])
		}
		b.WriteString("\n")
		for day := range grid {
			b.WriteString(rows[day])
			for slot := from; slot < to; slot++ {
				count := "·"
				switch {
				case closed[day][slot]:
					count = "×"
				case grid[day][slot] > 0:
					count = fmt.Sprint(grid[day][slot])
				}
				fmt.Fprintf(&b, " %2s", count)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("```")
	return b.String()
}

// halfDayTable renders the "Après-midi / Soir" table, unchanged from the original message.
func halfDayTable(m *i18n.Messages, rows []string, grid [][]int) string {
	var b strings.Builder
	b.WriteString("```md\n")
	fmt.Fprintf(&b, "%-10s %-11s %-11s\n", m.TableDayHeader, m.HalfDay[0], m.HalfDay[1])
	for day := range grid {
		fmt.Fprintf(&b, "%-10s", rows[day])
		for slot := range grid[day] {
			fmt.Fprintf(&b, " %-11d", grid[day][slot])
		}
		b.WriteString("\n")
	}
	b.WriteString("```")
	return b.String()
}
