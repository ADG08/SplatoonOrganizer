package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

// teamsPanel builds the list of the guild's teams.
func (h *HelpConfigHandler) teamsPanel(ctx context.Context, i *discordgo.InteractionCreate, status string) ([]discordgo.MessageComponent, error) {
	teams, m, err := GuildMessages(ctx, h.guilds, h.teams, i.GuildID, discord.LocaleMessages(i).Lang())
	if err != nil {
		return nil, err
	}
	return discord.BuildTeamsPanel(m, teams, status, h.supportURL), nil
}

// handleLanguage saves the guild's language picked in the setup, which the panel and the
// welcome message switch to.
func (h *HelpConfigHandler) handleLanguage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, value string) error {
	lang := i18n.Parse(value)
	if err := h.guilds.SetLanguage(ctx, i.GuildID, lang); err != nil {
		slog.ErrorContext(ctx, "help config: set guild language", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, discord.LocaleMessages(i).ErrGeneric)
	}
	slog.InfoContext(ctx, "guild language changed", "guild", i.GuildID, "by", discord.UserName(i), "language", lang)
	h.welcome.Refresh(s, i.GuildID)
	return h.showList(ctx, s, i, "")
}

// showList replaces the panel with the list of the guild's teams.
func (h *HelpConfigHandler) showList(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, status string) error {
	panel, err := h.teamsPanel(ctx, i, status)
	if err != nil {
		slog.ErrorContext(ctx, "help config: teams panel", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, discord.LocaleMessages(i).ErrGeneric)
	}
	return showPanel(s, i, panel)
}

// handleCreateSubmit creates a team, then goes on with its setup. It speaks the language
// of the panel it was created from.
func (h *HelpConfigHandler) handleCreateSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	_, m, err := GuildMessages(ctx, h.guilds, h.teams, i.GuildID, discord.LocaleMessages(i).Lang())
	if err != nil {
		slog.ErrorContext(ctx, "help config: guild texts", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, discord.LocaleMessages(i).ErrGeneric)
	}

	name := modalTextValue(i.ModalSubmitData().Components, "team_name")
	t, err := h.teams.Create(ctx, i.GuildID, name, m.Lang())
	if err != nil {
		return h.showList(ctx, s, i, "⚠️ "+teamErrorText(ctx, m, err))
	}
	logSettings(ctx, i, t, "team created")
	h.welcome.Refresh(s, i.GuildID)
	return showPanel(s, i, discord.BuildHelpPanel(t, fmt.Sprintf(m.TeamCreated, t.Name)))
}

// handleRenameSubmit renames a team. Its weekly message shows the name in its title.
func (h *HelpConfigHandler) handleRenameSubmit(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, t *team.Team, data discordgo.ModalSubmitInteractionData) error {
	m := t.Messages()
	if err := h.teams.Rename(ctx, t.ID, modalTextValue(data.Components, "team_name")); err != nil {
		return showPanel(s, i, discord.BuildHelpPanel(t, "⚠️ "+teamErrorText(ctx, m, err)))
	}
	renamed, err := h.teams.GetInGuild(ctx, i.GuildID, t.ID)
	if err != nil {
		slog.ErrorContext(ctx, "help config: get team", "guild", i.GuildID, "err", err)
		return discord.RespondError(s, i, m.ErrGeneric)
	}
	logSettings(ctx, i, t, "team renamed", "name", renamed.Name)
	h.refresher.Request(s, t.ID)
	return showPanel(s, i, discord.BuildHelpPanel(renamed, fmt.Sprintf(m.TeamRenamed, renamed.Name)))
}

// handlePublishNow posts the team's weekly message and reports how it went, which also
// resumes a paused team once it works.
func (h *HelpConfigHandler) handlePublishNow(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, t *team.Team) error {
	if err := deferPanelUpdate(s, i); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), weekly.Timeout)
	defer cancel()

	m := t.Messages()
	channel := "<#" + t.ChannelID + ">"
	if err := weekly.Publish(ctx, s, h.availSvc, t); err != nil {
		slog.ErrorContext(ctx, "publish weekly message", "guild", i.GuildID, "team", t.Name, "err", err)
		return editPanel(s, i, discord.BuildHelpPanel(t, "⚠️ "+fmt.Sprintf(m.PublishFailed, channel)))
	}
	if !t.Active {
		if err := h.teams.SetActive(ctx, t.ID, true); err != nil {
			slog.ErrorContext(ctx, "help config: resume team", "guild", i.GuildID, "err", err)
			return editPanel(s, i, discord.BuildHelpPanel(t, "⚠️ "+m.ErrGeneric))
		}
		t.Active = true
	}
	return editPanel(s, i, discord.BuildHelpPanel(t, fmt.Sprintf(m.PublishedIn, channel)))
}

// handleDelete deletes a team with its weekly message, then goes back to the list. The
// Discord channel and role stay: other things may use them.
func (h *HelpConfigHandler) handleDelete(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, t *team.Team) error {
	if err := deferPanelUpdate(s, i); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), weekly.Timeout)
	defer cancel()

	m := t.Messages()
	weekly.Remove(ctx, s, h.availSvc, t)
	if err := h.teams.Delete(ctx, t.ID); err != nil {
		slog.ErrorContext(ctx, "help config: delete team", "guild", i.GuildID, "err", err)
		return editPanel(s, i, discord.BuildHelpPanel(t, "⚠️ "+m.ErrGeneric))
	}
	logSettings(ctx, i, t, "team deleted")
	h.welcome.Refresh(s, i.GuildID)

	panel, err := h.teamsPanel(ctx, i, fmt.Sprintf(m.TeamDeleted, t.Name))
	if err != nil {
		slog.ErrorContext(ctx, "help config: teams panel", "guild", i.GuildID, "err", err)
		return nil // the panel keeps showing the deleted team, whose buttons say so
	}
	return editPanel(s, i, panel)
}

// teamErrorText explains why a team's name was refused.
func teamErrorText(ctx context.Context, m *i18n.Messages, err error) string {
	switch {
	case errors.Is(err, team.ErrNameEmpty):
		return m.NameEmpty
	case errors.Is(err, team.ErrNameTooLong):
		return fmt.Sprintf(m.NameTooLong, team.MaxNameLength)
	case errors.Is(err, team.ErrNameTaken):
		return m.NameTaken
	case errors.Is(err, team.ErrTooManyTeams):
		return fmt.Sprintf(m.TooManyTeams, team.MaxTeamsPerGuild)
	}
	slog.ErrorContext(ctx, "help config: save team", "err", err)
	return m.ErrGeneric
}

// modalTextValue returns what was typed in a form's text input, without surrounding spaces.
func modalTextValue(components []discordgo.MessageComponent, customID string) string {
	switch c := findModalComponent(components, customID).(type) {
	case discordgo.TextInput:
		return strings.TrimSpace(c.Value)
	case *discordgo.TextInput:
		return strings.TrimSpace(c.Value)
	}
	return ""
}
