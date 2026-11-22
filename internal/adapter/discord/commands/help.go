package commands

import (
	"context"
	"log/slog"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	appguild "github.com/ADG08/SplatoonOrganizer/internal/application/guild"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const CommandNameHelp = "help"

// HelpCommand handles the /help slash command for admin configuration.
type HelpCommand struct {
	teams      *appteam.Service
	guilds     *appguild.Service
	ownerID    string
	supportURL string
}

// NewHelpCommand creates a new help command. ownerID and supportURL are optional.
func NewHelpCommand(teams *appteam.Service, guilds *appguild.Service, ownerID, supportURL string) *HelpCommand {
	return &HelpCommand{teams: teams, guilds: guilds, ownerID: ownerID, supportURL: supportURL}
}

func (c *HelpCommand) Name() string {
	return CommandNameHelp
}

func (c *HelpCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	desc, descEN := description(func(m *i18n.Messages) string { return m.HelpDescription })
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameHelp,
		Description:              desc,
		DescriptionLocalizations: descEN,
	}
}

var _ discord.Command = (*HelpCommand)(nil)

func (c *HelpCommand) Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := discord.LocaleMessages(i)
	if !discord.IsAdminOrOwner(i, c.ownerID) {
		return discord.RespondError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return discord.RespondError(s, i, m.ErrGuildOnly)
	}

	teams, gm, err := handlers.GuildMessages(ctx, c.guilds, c.teams, i.GuildID, m.Lang())
	if err != nil {
		slog.ErrorContext(ctx, "help: guild texts", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	m = gm

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
			Components: discord.BuildTeamsPanel(m, teams, "", c.supportURL),
		},
	})
}
