package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const CommandNameAnnonce = "annonce"

// AnnonceCommand handles the /annonce slash command for publishing an embed announcement.
type AnnonceCommand struct {
	svc     *appguildconfig.Service
	ownerID string
}

// NewAnnonceCommand creates a new annonce command. ownerID is optional.
func NewAnnonceCommand(svc *appguildconfig.Service, ownerID string) *AnnonceCommand {
	return &AnnonceCommand{svc: svc, ownerID: ownerID}
}

func (c *AnnonceCommand) Name() string {
	return CommandNameAnnonce
}

func (c *AnnonceCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	option := func(kind discordgo.ApplicationCommandOptionType, name string, required bool, text func(*i18n.Messages) string) *discordgo.ApplicationCommandOption {
		desc, descEN := description(text)
		return &discordgo.ApplicationCommandOption{Type: kind, Name: name, Description: desc, DescriptionLocalizations: *descEN, Required: required}
	}
	desc, descEN := description(func(m *i18n.Messages) string { return m.AnnonceDescription })
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameAnnonce,
		Description:              desc,
		DescriptionLocalizations: descEN,
		Options: []*discordgo.ApplicationCommandOption{
			option(discordgo.ApplicationCommandOptionString, "titre", true, func(m *i18n.Messages) string { return m.AnnonceTitleDesc }),
			option(discordgo.ApplicationCommandOptionString, "message", true, func(m *i18n.Messages) string { return m.AnnonceMessageDesc }),
			option(discordgo.ApplicationCommandOptionRole, "role", false, func(m *i18n.Messages) string { return m.AnnonceRoleDesc }),
		},
	}
}

var _ discord.Command = (*AnnonceCommand)(nil)

func (c *AnnonceCommand) Execute(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cfg, err := c.svc.Get(ctx, i.GuildID)
	if err != nil {
		slog.ErrorContext(ctx, "annonce: get guild config", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, i18n.For(i18n.FR).ErrGeneric)
	}
	m := cfg.Messages()
	if !discord.IsAdminOrOwner(i, c.ownerID) {
		return discord.RespondError(s, i, m.ErrAdminOnly)
	}
	if i.GuildID == "" {
		return discord.RespondError(s, i, m.ErrGuildOnly)
	}

	channelID := cfg.ChannelID
	if channelID == "" {
		return discord.RespondError(s, i, m.NoChannel)
	}

	opts := newOptionMap(i.ApplicationCommandData().Options)
	titre := strings.TrimSpace(opts.str("titre"))
	message := opts.str("message")
	roleID := opts.roleID("role")

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

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(m.AnnouncePublished, "<#"+channelID+">"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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
