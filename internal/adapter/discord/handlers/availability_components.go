package handlers

import (
	"fmt"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

// buildUserDisposSelectMenus builds one select menu per day for days [dayStart, dayStart+dayCount).
func buildUserDisposSelectMenus(userSlots map[int]map[int]bool, dayStart, dayCount int) []discordgo.MessageComponent {
	dayPlaceholders := []string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"}
	slotOptions := []struct {
		Value string
		Label string
	}{
		{"1", "Après-midi"},
		{"2", "Soir"},
	}

	minVals := 0
	maxVals := 2

	var rows []discordgo.MessageComponent
	for day := dayStart; day < dayStart+dayCount && day < availability.DaysPerWeek; day++ {
		daySlots := userSlots[day]
		options := make([]discordgo.SelectMenuOption, availability.SlotsPerDay)
		for slot := 0; slot < availability.SlotsPerDay; slot++ {
			available := daySlots != nil && daySlots[slot]
			options[slot] = discordgo.SelectMenuOption{
				Label:   slotOptions[slot].Label,
				Value:   slotOptions[slot].Value,
				Default: available,
			}
		}
		menu := discordgo.SelectMenu{
			CustomID:    fmt.Sprintf("%s:%d", SelectDisposCustomIDPrefix, day),
			Placeholder: dayPlaceholders[day],
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
