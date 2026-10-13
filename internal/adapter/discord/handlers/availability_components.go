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
// option text of a slot, in the member's timezone.
func buildUserDisposSelectMenus(m *i18n.Messages, layout availability.Layout, userSlots map[int]map[int]bool, dayStart, dayCount int, label func(day, slot int) string) []discordgo.MessageComponent {
	minVals := 0
	maxVals := layout.Len()

	var rows []discordgo.MessageComponent
	for day := dayStart; day < dayStart+dayCount && day < availability.DaysPerWeek; day++ {
		daySlots := userSlots[day]
		options := make([]discordgo.SelectMenuOption, layout.Len())
		for slot := 0; slot < layout.Len(); slot++ {
			available := daySlots != nil && daySlots[slot]
			options[slot] = discordgo.SelectMenuOption{
				Label:   label(day, slot),
				Value:   strconv.Itoa(slot + 1),
				Default: available,
			}
		}
		menu := discordgo.SelectMenu{
			CustomID:    fmt.Sprintf("%s:%d", SelectDisposCustomIDPrefix, day),
			Placeholder: m.Days[day],
			MinValues:   &minVals,
			MaxValues:   maxVals,
			Options:     options,
		}
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})
	}
	return rows
}
