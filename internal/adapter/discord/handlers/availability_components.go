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
// publication) are left out; a day without any open slot gets a disabled menu.
func buildUserDisposSelectMenus(m *i18n.Messages, layout availability.Layout, userSlots map[int]map[int]bool, closed [][]bool, dayStart, dayCount int, label func(day, slot int) string) []discordgo.MessageComponent {
	minVals := 0

	var rows []discordgo.MessageComponent
	for day := dayStart; day < dayStart+dayCount && day < availability.DaysPerWeek; day++ {
		daySlots := userSlots[day]
		options := make([]discordgo.SelectMenuOption, 0, layout.Len())
		for slot := range layout.Len() {
			if isClosed(closed, day, slot) {
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
			menu.Options = []discordgo.SelectMenuOption{{Label: "—", Value: "0"}}
			menu.MaxValues = 1
		}
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})
	}
	return rows
}

// isClosed reports whether a slot is closed; a nil closed means none is.
func isClosed(closed [][]bool, day, slot int) bool {
	return day < len(closed) && slot < len(closed[day]) && closed[day][slot]
}
