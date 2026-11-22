package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const CommandNameAnnonce = "annonce"

// maxAutocompleteChoices is Discord's limit of choices an autocomplete can offer.
const maxAutocompleteChoices = 25

// AnnonceCommand handles the /annonce slash command for publishing an embed announcement
// in a team's channel.
type AnnonceCommand struct {
	teams   *appteam.Service
	ownerID string
}

// NewAnnonceCommand creates a new annonce command. ownerID is optional.
func NewAnnonceCommand(teams *appteam.Service, ownerID string) *AnnonceCommand {
	return &AnnonceCommand{teams: teams, ownerID: ownerID}
}

func (c *AnnonceCommand) Name() string {
	return CommandNameAnnonce
}

func (c *AnnonceCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	option := func(kind discordgo.ApplicationCommandOptionType, name string, required bool, text func(*i18n.Messages) string) *discordgo.ApplicationCommandOption {
		desc, descEN := description(text)
		return &discordgo.ApplicationCommandOption{Type: kind, Name: name, Description: desc, DescriptionLocalizations: *descEN, Required: required}
	}
	equipe := option(discordgo.ApplicationCommandOptionString, "equipe", false, func(m *i18n.Messages) string { return m.AnnonceTeamDesc })
	equipe.Autocomplete = true
	desc, descEN := description(func(m *i18n.Messages) string { return m.AnnonceDescription })
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameAnnonce,
		Description:              desc,
		DescriptionLocalizations: descEN,
		Options: []*discordgo.ApplicationCommandOption{
			option(discordgo.ApplicationCommandOptionString, "titre", true, func(m *i18n.Messages) string { return m.AnnonceTitleDesc }),
			option(discordgo.ApplicationCommandOptionString, "message", true, func(m *i18n.Messages) string { return m.AnnonceMessageDesc }),
			equipe,
			option(discordgo.ApplicationCommandOptionRole, "role", false, func(m *i18n.Messages) string { return m.AnnonceRoleDesc }),
		},
	}
}

var (
	_ discord.Command       = (*AnnonceCommand)(nil)
	_ discord.Autocompleter = (*AnnonceCommand)(nil)
)

func (c *AnnonceCommand) Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := discord.LocaleMessages(i)
	if !discord.IsAdminOrOwner(i, c.ownerID) {
		return discord.RespondError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return discord.RespondError(s, i, m.ErrGuildOnly)
	}

	opts := newOptionMap(i.ApplicationCommandData().Options)
	t, problem := c.resolveTeam(ctx, i, m, strings.TrimSpace(opts.str("equipe")))
	if t == nil {
		return discord.RespondError(s, i, problem)
	}
	m = t.Messages()
	channelID := t.ChannelID
	if channelID == "" {
		return discord.RespondError(s, i, fmt.Sprintf(m.NoChannel, t.Name))
	}

	titre := strings.TrimSpace(opts.str("titre"))
	message := opts.str("message")
	roleID := opts.roleID("role")
	if roleID == "" {
		roleID = t.RoleID
	}

	if titre == "" || strings.TrimSpace(message) == "" {
		return discord.RespondError(s, i, m.AnnounceEmpty)
	}

	// Allow line breaks via the literal "\n" typed in the slash command option.
	message = strings.ReplaceAll(message, "\\n", "\n")

	body := message
	if i.Member != nil && i.Member.User != nil {
		// A mention pill is clickable and opens the Discord profile card (embed mentions never ping).
		body += "\n\n" + fmt.Sprintf(m.AnnounceBy, "<@"+i.Member.User.ID+">")
	}

	embed := &discordgo.MessageEmbed{
		Title:       titre,
		Description: body,
		Color:       discord.HelpPanelColorDefault,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	send := &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}}
	if roleID != "" {
		send.Content = "<@&" + roleID + ">"
		send.AllowedMentions = &discordgo.MessageAllowedMentions{
			Roles: []string{roleID},
		}
	}

	if _, err := s.ChannelMessageSendComplex(channelID, send); err != nil {
		slog.ErrorContext(ctx, "annonce: send message", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	slog.InfoContext(ctx, "announcement sent", "guild", i.GuildID, "team", t.Name, "by", discord.UserName(i), "title", titre)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(m.AnnouncePublished, "<#"+channelID+">"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// Autocomplete suggests the guild's teams for the "equipe" option.
func (c *AnnonceCommand) Autocomplete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	var typed string
	for _, o := range i.ApplicationCommandData().Options {
		if o.Focused {
			typed = strings.ToLower(strings.TrimSpace(o.StringValue()))
		}
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{}
	if i.GuildID != "" {
		teams, err := c.teams.List(ctx, i.GuildID)
		if err != nil {
			slog.ErrorContext(ctx, "annonce: list teams", "guild", i.GuildID, "err", err)
		}
		for _, t := range teams {
			if len(choices) < maxAutocompleteChoices && strings.Contains(strings.ToLower(t.Name), typed) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: t.Name, Value: strconv.FormatInt(t.ID, 10)})
			}
		}
	}
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

// resolveTeam finds the team of the "equipe" option, an autocompleted ID or a typed name;
// without it, the guild's only team. It returns nil and the reason when there is none.
func (c *AnnonceCommand) resolveTeam(ctx context.Context, i *discordgo.InteractionCreate, m *i18n.Messages, value string) (*team.Team, string) {
	teams, err := c.teams.List(ctx, i.GuildID)
	if err != nil {
		slog.ErrorContext(ctx, "annonce: list teams", "guild", i.GuildID, "err", err)
		return nil, m.ErrGeneric
	}
	if len(teams) == 0 {
		return nil, m.NoTeam
	}
	if value == "" {
		if len(teams) == 1 {
			return teams[0], ""
		}
		return nil, m.PickTeam
	}
	for _, t := range teams {
		if strconv.FormatInt(t.ID, 10) == value || strings.EqualFold(t.Name, value) {
			return t, ""
		}
	}
	return nil, fmt.Sprintf(m.UnknownTeam, value)
}

// optionMap indexes slash command options by name for convenient lookup.
type optionMap map[string]*discordgo.ApplicationCommandInteractionDataOption

func newOptionMap(opts []*discordgo.ApplicationCommandInteractionDataOption) optionMap {
	m := make(optionMap, len(opts))
	for _, o := range opts {
		m[o.Name] = o
	}
	return m
}

func (m optionMap) str(name string) string {
	if o, ok := m[name]; ok {
		return o.StringValue()
	}
	return ""
}

// roleID returns the selected role ID for a Role option, or "" if absent.
func (m optionMap) roleID(name string) string {
	if o, ok := m[name]; ok {
		if id, ok := o.Value.(string); ok {
			return id
		}
	}
	return ""
}
