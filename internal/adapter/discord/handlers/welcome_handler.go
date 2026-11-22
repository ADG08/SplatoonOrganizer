package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appguild "github.com/ADG08/SplatoonOrganizer/internal/application/guild"
	appteam "github.com/ADG08/SplatoonOrganizer/internal/application/team"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guild"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// welcomeTimeout bounds what is done for the welcome message, outside of any interaction.
const welcomeTimeout = 10 * time.Second

// botPostPermissions are what the bot needs to post its embeds in a channel.
const botPostPermissions = discordgo.PermissionViewChannel | discordgo.PermissionSendMessages | discordgo.PermissionEmbedLinks

// GuildMessages returns the guild's teams, with the texts of what concerns the whole guild
// (the list of its teams, its welcome message): see discord.GuildLang.
func GuildMessages(ctx context.Context, guilds *appguild.Service, teams *appteam.Service, guildID string, fallback i18n.Lang) ([]*team.Team, *i18n.Messages, error) {
	list, err := teams.List(ctx, guildID)
	if err != nil {
		return nil, nil, fmt.Errorf("list teams: %w", err)
	}
	g, err := guilds.Get(ctx, guildID)
	if err != nil {
		return nil, nil, fmt.Errorf("get guild: %w", err)
	}
	return list, i18n.For(discord.GuildLang(g.Language, list, fallback)), nil
}

// WelcomeHandler posts the message welcoming the bot in a guild, keeps its checklist up
// to date as the setup goes on, and handles its buttons.
type WelcomeHandler struct {
	guilds     *appguild.Service
	teams      *appteam.Service
	ownerID    string
	supportURL string
}

// NewWelcomeHandler creates a new welcome handler. ownerID and supportURL are optional.
func NewWelcomeHandler(guilds *appguild.Service, teams *appteam.Service, ownerID, supportURL string) *WelcomeHandler {
	return &WelcomeHandler{guilds: guilds, teams: teams, ownerID: ownerID, supportURL: supportURL}
}

func (h *WelcomeHandler) CustomID() string {
	return discord.WelcomeCustomIDPrefix
}

var _ discord.InteractionHandler = (*WelcomeHandler)(nil)

func (h *WelcomeHandler) Handle(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := discord.LocaleMessages(i)
	customID := i.MessageComponentData().CustomID
	if lang, ok := strings.CutPrefix(customID, discord.WelcomeLanguageCustomID); ok {
		return h.pickLanguage(ctx, s, i, lang)
	}
	teams, gm, err := GuildMessages(ctx, h.guilds, h.teams, i.GuildID, m.Lang())
	if err != nil {
		slog.ErrorContext(ctx, "welcome: guild texts", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	switch customID {
	case discord.WelcomeSetupCustomID:
		if !discord.IsAdminOrOwner(i, h.ownerID) {
			return discord.RespondError(s, i, m.ErrAdminOnly)
		}
		m = gm
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:      discordgo.MessageFlagsEphemeral | discordgo.MessageFlagsIsComponentsV2,
				Components: discord.BuildTeamsPanel(m, teams, "", h.supportURL),
			},
		})
	case discord.WelcomeHowToCustomID:
		m = gm // the language of the message clicked
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{{Title: m.HowToTitle, Description: m.HowToBody, Color: discord.HelpPanelColorDefault}},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
	}
	return discord.RespondError(s, i, m.ErrInvalid)
}

// pickLanguage switches the guild's language from the welcome message, which follows it.
func (h *WelcomeHandler) pickLanguage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, value string) error {
	m := discord.LocaleMessages(i)
	if !discord.IsAdminOrOwner(i, h.ownerID) {
		return discord.RespondError(s, i, m.ErrAdminOnly)
	}
	lang := i18n.Parse(value)
	if err := h.guilds.SetLanguage(ctx, i.GuildID, lang); err != nil {
		slog.ErrorContext(ctx, "welcome: set language", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	slog.InfoContext(ctx, "guild language changed", "guild", i.GuildID, "by", discord.UserName(i), "language", lang)
	teams, err := h.teams.List(ctx, i.GuildID)
	if err != nil {
		slog.ErrorContext(ctx, "welcome: list teams", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	embed, components := discord.BuildWelcome(i18n.For(lang), teams, s.State.User)
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components},
	})
}

// Welcome posts the welcome message in a guild the bot just joined, unless it has teams
// or already has one (a restart right after the join announces the guild as new again).
func (h *WelcomeHandler) Welcome(s *discordgo.Session, g *discordgo.Guild) {
	ctx, cancel := context.WithTimeout(context.Background(), welcomeTimeout)
	defer cancel()

	saved, err := h.guilds.Get(ctx, g.ID)
	if err != nil {
		slog.ErrorContext(ctx, "welcome: get guild", "guild", g.ID, "err", err)
		return
	}
	teams, err := h.teams.List(ctx, g.ID)
	if err != nil {
		slog.ErrorContext(ctx, "welcome: list teams", "guild", g.ID, "err", err)
		return
	}
	if saved.Welcome != nil || len(teams) > 0 {
		return
	}

	channelID := welcomeChannel(s, g)
	if channelID == "" {
		slog.WarnContext(ctx, "welcome: no channel to post in", "guild", g.ID)
		return
	}
	// In French until an admin switches it: a guild that had the bot before keeps its choice.
	embed, components := discord.BuildWelcome(i18n.For(discord.GuildLang(saved.Language, nil, i18n.FR)), nil, s.State.User)
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{embed},
		Components:      components,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
	}, discordgo.WithContext(ctx))
	if err != nil {
		slog.ErrorContext(ctx, "welcome: send message", "guild", g.ID, "err", err)
		return
	}
	if err := h.guilds.SetWelcome(ctx, g.ID, guild.Welcome{ChannelID: channelID, MessageID: msg.ID}); err != nil {
		slog.ErrorContext(ctx, "welcome: save message", "guild", g.ID, "err", err)
	}
}

// Forget forgets the guild's welcome message: the bot left, it is welcomed again if it
// comes back without teams.
func (h *WelcomeHandler) Forget(guildID string) {
	ctx, cancel := context.WithTimeout(context.Background(), welcomeTimeout)
	defer cancel()
	if err := h.guilds.ForgetWelcome(ctx, guildID); err != nil {
		slog.ErrorContext(ctx, "welcome: forget message", "guild", guildID, "err", err)
	}
}

// Refresh updates the welcome message's checklist (best-effort), in the background: it
// doesn't hold the answer of the interaction that changed the setup.
func (h *WelcomeHandler) Refresh(s *discordgo.Session, guildID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), welcomeTimeout)
		defer cancel()

		g, err := h.guilds.Get(ctx, guildID)
		if err != nil || g.Welcome == nil {
			if err != nil {
				slog.ErrorContext(ctx, "welcome: get guild", "guild", guildID, "err", err)
			}
			return
		}
		welcome := g.Welcome
		teams, err := h.teams.List(ctx, guildID)
		if err != nil {
			slog.ErrorContext(ctx, "welcome: list teams", "guild", guildID, "err", err)
			return
		}
		embed, components := discord.BuildWelcome(i18n.For(discord.GuildLang(g.Language, teams, i18n.FR)), teams, s.State.User)
		if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         welcome.MessageID,
			Channel:    welcome.ChannelID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}, discordgo.WithContext(ctx)); err != nil {
			slog.ErrorContext(ctx, "welcome: edit message", "guild", guildID, "err", err)
		}
	}()
}

// welcomeChannel returns the guild's system channel if the bot can post there, else the
// first text channel where it can.
func welcomeChannel(s *discordgo.Session, g *discordgo.Guild) string {
	if g.SystemChannelID != "" && botCanPost(s, g.SystemChannelID) {
		return g.SystemChannelID
	}
	var text []*discordgo.Channel
	for _, c := range g.Channels {
		if c.Type == discordgo.ChannelTypeGuildText {
			text = append(text, c)
		}
	}
	slices.SortFunc(text, func(a, b *discordgo.Channel) int { return a.Position - b.Position })
	for _, c := range text {
		if botCanPost(s, c.ID) {
			return c.ID
		}
	}
	return ""
}

// botCanPost reports whether the bot can post its embeds in a channel, as far as it knows.
func botCanPost(s *discordgo.Session, channelID string) bool {
	perms, err := s.State.UserChannelPermissions(s.State.User.ID, channelID)
	return err == nil && perms&botPostPermissions == botPostPermissions
}
