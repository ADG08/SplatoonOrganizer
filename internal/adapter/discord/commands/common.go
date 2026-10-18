package commands

import (
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// description returns a command text in French (the default) with its English
// localization: Discord shows command descriptions in each member's app language, as
// global commands can't follow a guild setting.
func description(text func(*i18n.Messages) string) (string, *map[discordgo.Locale]string) {
	en := text(i18n.For(i18n.EN))
	return text(i18n.For(i18n.FR)), &map[discordgo.Locale]string{discordgo.EnglishUS: en, discordgo.EnglishGB: en}
}
