package handlers

import (
	"fmt"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// buildUserDisposSelectMenus builds one select menu per day for days [dayStart, dayStart+dayCount).
// Option values are 1-based slot indexes ("1" = first slot of the layout); label gives the
// option text of a slot, in the member's timezone. Closed slots (before the week's
// publication) and past ones are left out; a day without any slot left gets a disabled menu.
func buildUserDisposSelectMenus(m *i18n.Messages, layout availability.Layout, userSlots map[int]map[int]bool, closed, past [][]bool, dayStart, dayCount int, label func(day, slot int) string) []discordgo.MessageComponent {
	minVals := 0

	var rows []discordgo.MessageComponent
	for day := dayStart; day < dayStart+dayCount && day < availability.DaysPerWeek; day++ {
		daySlots := userSlots[day]
		options := make([]discordgo.SelectMenuOption, 0, layout.Len())
		for slot := range layout.Len() {
			if marked(closed, day, slot) || marked(past, day, slot) {
				continue
			}
			options = append(options, discordgo.SelectMenuOption{
				Label:   label(day, slot),
				Value:   strconv.Itoa(slot + 1),
				Default: daySlots != nil && daySlots[slot],
			})
		}
		menu := discordgo.SelectMenu{
			CustomID:    fmt.Sprintf("%s:%d", SelectDisposCustomIDPrefix, day),
			Placeholder: m.Days[day],
			MinValues:   &minVals,
			MaxValues:   len(options),
			Options:     options,
		}
		if len(options) == 0 {
			// Discord wants an option even in a disabled menu.
			menu.Disabled = true
			menu.Placeholder = fmt.Sprintf(m.DayClosed, m.Days[day])
			if marked(past, day, 0) {
				menu.Placeholder = fmt.Sprintf(m.DayPast, m.Days[day])
			}
			menu.Options = []discordgo.SelectMenuOption{{Label: "—", Value: "0"}}
			menu.MaxValues = 1
		}
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})
	}
	return rows
}

// marked reports whether a slot is marked in a grid of slots; a nil grid marks none.
func marked(grid [][]bool, day, slot int) bool {
	return day < len(grid) && slot < len(grid[day]) && grid[day][slot]
}
