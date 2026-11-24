// Package weekly builds, publishes and refreshes each team's weekly availability
// message. Its components are answered by package handlers.
package weekly

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/gridimage"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const (
	// AccentColor is the color of the weekly message, reused by the ephemeral answers.
	AccentColor = 0xFEE75C
	// ImageName is the attachment holding the grid image of the "image" display.
	ImageName = "dispos.png"
)

// Payload is the weekly availability message for one display, ready to be sent,
// applied as an edit, or used as an interaction response.
type Payload struct {
	Content    string
	Embeds     []*discordgo.MessageEmbed
	Components []discordgo.MessageComponent
	Files      []*discordgo.File
	Flags      discordgo.MessageFlags
	// AllowedMentions only lets the configured role be pinged.
	AllowedMentions *discordgo.MessageAllowedMentions
}

// Title adds the team's name to a title: a guild can have several weekly messages.
func Title(t *team.Team, title string) string {
	return title + " · " + t.Name
}

// BuildPayload builds the team's weekly message, in its display. When the team has a
// role, the message mentions it to ping it.
func BuildPayload(t *team.Team, view appavailability.WeeklyView) (*Payload, error) {
	m, roleID, display := t.Messages(), t.RoleID, t.Display
	p := &Payload{AllowedMentions: &discordgo.MessageAllowedMentions{}}
	if roleID != "" {
		p.AllowedMentions.Roles = []string{roleID}
	}

	if display != team.DisplayImage {
		p.Embeds = []*discordgo.MessageEmbed{buildClassicEmbed(t, view)}
		p.Components = buildClassicComponents(m, t.ID, t.MemberTime())
		if roleID != "" {
			p.Content = "<@&" + roleID + ">"
		}
		return p, nil
	}

	img, err := gridimage.Render(view, t.FullAt)
	if err != nil {
		return nil, fmt.Errorf("render grid image: %w", err)
	}
	p.Files = []*discordgo.File{{Name: ImageName, ContentType: "image/png", Reader: bytes.NewReader(img)}}
	p.Flags = discordgo.MessageFlagsIsComponentsV2
	if roleID != "" {
		p.Components = append(p.Components, discordgo.TextDisplay{Content: "<@&" + roleID + ">"})
	}
	p.Components = append(p.Components, buildImageContainer(t, view))
	return p, nil
}

// MessageSend converts the payload into a channel message.
func (p *Payload) MessageSend() *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Content:         p.Content,
		Embeds:          p.Embeds,
		Components:      p.Components,
		Files:           p.Files,
		Flags:           p.Flags,
		AllowedMentions: p.AllowedMentions,
	}
}

// MessageEdit converts the payload into an edit of an existing weekly message
// of the same display. The grid image, if any, replaces the previous attachment.
func (p *Payload) MessageEdit(channelID, messageID string) *discordgo.MessageEdit {
	edit := &discordgo.MessageEdit{
		ID:              messageID,
		Channel:         channelID,
		Components:      &p.Components,
		Flags:           p.Flags,
		AllowedMentions: p.AllowedMentions,
	}
	if p.Embeds != nil {
		edit.Embeds = &p.Embeds
	}
	if p.Files != nil {
		// Keep only the newly uploaded files: attachment ids of new uploads are their index.
		attachments := make([]*discordgo.MessageAttachment, len(p.Files))
		for i, f := range p.Files {
			attachments[i] = &discordgo.MessageAttachment{ID: strconv.Itoa(i), Filename: f.Name}
		}
		edit.Files = p.Files
		edit.Attachments = &attachments
	}
	return edit
}

// buildClassicEmbed is the original embed: the text table, plus the timezone hint for
// hourly layouts.
func buildClassicEmbed(t *team.Team, view appavailability.WeeklyView) *discordgo.MessageEmbed {
	description := view.Table
	if view.Legend != "" {
		description = "*" + view.Legend + "*\n" + view.Table
	}
	return &discordgo.MessageEmbed{
		Title:       Title(t, t.Messages().ClassicTitle),
		Description: description,
		Color:       AccentColor,
	}
}

// buildClassicComponents returns the action row with the original weekly buttons,
// plus "Mon fuseau" when members see the slots in their own timezone.
func buildClassicComponents(m *i18n.Messages, teamID int64, memberTime bool) []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    m.BtnMyDispos,
			Style:    discordgo.PrimaryButton,
			CustomID: CustomID(OpenDisposCustomID, teamID),
		},
		discordgo.Button{
			Label:    m.BtnViewDispos,
			Style:    discordgo.SecondaryButton,
			CustomID: CustomID(ViewDisposCustomID, teamID),
		},
		discordgo.Button{
			Label:    m.BtnUnavailable,
			Style:    discordgo.SecondaryButton,
			CustomID: CustomID(UnavailableCustomID, teamID),
		},
	}
	if memberTime {
		buttons = append(buttons, myTimeButton(m, teamID))
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

// myTimeButton shows the week's grid in the member's own timezone.
func myTimeButton(m *i18n.Messages, teamID int64) discordgo.Button {
	return discordgo.Button{
		Label:    m.BtnMyTimezone,
		Style:    discordgo.SecondaryButton,
		CustomID: CustomID(MyTimeCustomID, teamID),
	}
}

// buildImageContainer lays out the "image" display: title, grid image, actions.
func buildImageContainer(t *team.Team, view appavailability.WeeklyView) discordgo.Container {
	m := t.Messages()
	legend := m.ColorLegend(t.FullAt)
	if view.Legend != "" {
		legend = view.Legend + " · " + legend
	}
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    m.BtnMyDispos,
			Style:    discordgo.PrimaryButton,
			CustomID: CustomID(OpenDisposCustomID, t.ID),
		},
		discordgo.Button{
			Label:    m.BtnUnavailable,
			Style:    discordgo.SecondaryButton,
			CustomID: CustomID(UnavailableCustomID, t.ID),
		},
	}
	if t.MemberTime() {
		buttons = append(buttons, myTimeButton(m, t.ID))
	}

	accent := AccentColor
	return discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## " + Title(t, view.Title) + "\n-# " + legend},
			discordgo.MediaGallery{
				Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://" + ImageName}}},
			},
			discordgo.ActionsRow{Components: buttons},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{buildWhoDisposSelect(m, t.ID)},
			},
		},
	}
}

// buildWhoDisposSelect lets members see who is available on a given day (answered in ephemeral).
// Discord has no clickable grid, so this select replaces "clicking a cell".
func buildWhoDisposSelect(m *i18n.Messages, teamID int64) discordgo.SelectMenu {
	options := []discordgo.SelectMenuOption{
		{Label: m.WhoWholeWeek, Value: WhoAllValue, Emoji: &discordgo.ComponentEmoji{Name: "📋"}},
	}
	for day := range availability.DaysPerWeek {
		options = append(options, discordgo.SelectMenuOption{
			Label: m.Days[day],
			Value: strconv.Itoa(day),
		})
	}
	return discordgo.SelectMenu{
		CustomID:    CustomID(WhoDisposCustomID, teamID),
		Placeholder: m.WhoPlaceholder,
		Options:     options,
	}
}
