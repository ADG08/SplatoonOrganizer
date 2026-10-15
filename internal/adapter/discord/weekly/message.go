// Package weekly builds, publishes and refreshes each guild's weekly availability
// message. Its components are answered by package handlers.
package weekly

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/gridimage"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
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

// BuildPayload builds the weekly message. If roleID is not empty, the message
// mentions the role (<@&roleID>) to ping it.
func BuildPayload(m *i18n.Messages, view appavailability.WeeklyView, roleID string, display guildconfig.Display) (*Payload, error) {
	p := &Payload{AllowedMentions: &discordgo.MessageAllowedMentions{}}
	if roleID != "" {
		p.AllowedMentions.Roles = []string{roleID}
	}

	if display != guildconfig.DisplayImage {
		p.Embeds = []*discordgo.MessageEmbed{buildClassicEmbed(m, view)}
		p.Components = buildClassicComponents(m, view.Hourly)
		if roleID != "" {
			p.Content = "<@&" + roleID + ">"
		}
		return p, nil
	}

	img, err := gridimage.Render(view)
	if err != nil {
		return nil, fmt.Errorf("render grid image: %w", err)
	}
	p.Files = []*discordgo.File{{Name: ImageName, ContentType: "image/png", Reader: bytes.NewReader(img)}}
	p.Flags = discordgo.MessageFlagsIsComponentsV2
	if roleID != "" {
		p.Components = append(p.Components, discordgo.TextDisplay{Content: "<@&" + roleID + ">"})
	}
	p.Components = append(p.Components, buildImageContainer(m, view))
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
func buildClassicEmbed(m *i18n.Messages, view appavailability.WeeklyView) *discordgo.MessageEmbed {
	description := view.Table
	if view.Legend != "" {
		description = "*" + view.Legend + "*\n" + view.Table
	}
	return &discordgo.MessageEmbed{
		Title:       m.ClassicTitle,
		Description: description,
		Color:       AccentColor,
	}
}

// buildClassicComponents returns the action row with the original weekly buttons,
// plus "Mon fuseau" when the slots are hours.
func buildClassicComponents(m *i18n.Messages, hourly bool) []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    m.BtnMyDispos,
			Style:    discordgo.PrimaryButton,
			CustomID: OpenDisposCustomID,
		},
		discordgo.Button{
			Label:    m.BtnViewDispos,
			Style:    discordgo.SecondaryButton,
			CustomID: ViewDisposCustomID,
		},
		discordgo.Button{
			Label:    m.BtnUnavailable,
			Style:    discordgo.SecondaryButton,
			CustomID: UnavailableCustomID,
		},
	}
	if hourly {
		buttons = append(buttons, myTimeButton(m))
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

// myTimeButton shows the week's grid in the member's own timezone.
func myTimeButton(m *i18n.Messages) discordgo.Button {
	return discordgo.Button{
		Label:    m.BtnMyTimezone,
		Style:    discordgo.SecondaryButton,
		CustomID: MyTimeCustomID,
	}
}

// buildImageContainer lays out the "image" display: title, grid image, actions.
func buildImageContainer(m *i18n.Messages, view appavailability.WeeklyView) discordgo.Container {
	legend := m.ImageLegend
	if view.Legend != "" {
		legend = view.Legend + " · " + legend
	}
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    m.BtnMyDispos,
			Style:    discordgo.PrimaryButton,
			CustomID: OpenDisposCustomID,
		},
		discordgo.Button{
			Label:    m.BtnUnavailable,
			Style:    discordgo.SecondaryButton,
			CustomID: UnavailableCustomID,
		},
	}
	if view.Hourly {
		buttons = append(buttons, myTimeButton(m))
	}

	accent := AccentColor
	return discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## " + view.Title + "\n-# " + legend},
			discordgo.MediaGallery{
				Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://" + ImageName}}},
			},
			discordgo.ActionsRow{Components: buttons},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{buildWhoDisposSelect(m)},
			},
		},
	}
}

// buildWhoDisposSelect lets members see who is available on a given day (answered in ephemeral).
// Discord has no clickable grid, so this select replaces "clicking a cell".
func buildWhoDisposSelect(m *i18n.Messages) discordgo.SelectMenu {
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
		CustomID:    WhoDisposCustomID,
		Placeholder: m.WhoPlaceholder,
		Options:     options,
	}
}
