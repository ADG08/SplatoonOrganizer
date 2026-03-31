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

// BuildWeeklyMessage builds the full message (embed + edit button).
// If roleID is not empty, the message includes a role mention (<@&roleID>) for pinging.
func BuildWeeklyMessage(table string, roleID string) *discordgo.MessageSend {
	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			BuildWeeklyEmbed(table),
		},
		Components: []discordgo.MessageComponent{
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
		},
	}
	if roleID != "" {
		msg.Content = "<@&" + roleID + ">"
	}
	return msg
}
