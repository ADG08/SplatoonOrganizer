package handlers

import "github.com/bwmarrin/discordgo"

// BuildWeeklyEmbed builds the main embed displaying availability summary.
func BuildWeeklyEmbed(table string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Disponibilités de la semaine",
		Description: table,
		Color:       0xFEE75C,
	}
}

// BuildWeeklyComponents returns the action row with the weekly availability buttons.
func BuildWeeklyComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "✏️ Mes dispos",
					Style:    discordgo.PrimaryButton,
					CustomID: OpenDisposCustomIDPrefix,
				},
				discordgo.Button{
					Label:    "📋 Voir les dispos",
					Style:    discordgo.SecondaryButton,
					CustomID: ViewDisposCustomIDPrefix,
				},
			},
		},
	}
}

// BuildWeeklyMessage builds the full message (embed + buttons).
// If roleID is not empty, the message includes a role mention (<@&roleID>) for pinging.
func BuildWeeklyMessage(table string, roleID string) *discordgo.MessageSend {
	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{BuildWeeklyEmbed(table)},
		Components: BuildWeeklyComponents(),
	}
	if roleID != "" {
		msg.Content = "<@&" + roleID + ">"
	}
	return msg
}
