package handlers

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/gridimage"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

const (
	weeklyAccentColor = 0xFEE75C
	// weeklyImageName is the attachment holding the grid image of the "image" display.
	weeklyImageName = "dispos.webp"
	// whoDisposAllValue is the "who is available" option showing the whole week.
	whoDisposAllValue = "all"
)

// WeeklyPayload is the weekly availability message for one display, ready to be sent,
// applied as an edit, or used as an interaction response.
type WeeklyPayload struct {
	Content    string
	Embeds     []*discordgo.MessageEmbed
	Components []discordgo.MessageComponent
	Files      []*discordgo.File
	Flags      discordgo.MessageFlags
	// AllowedMentions only lets the configured role be pinged.
	AllowedMentions *discordgo.MessageAllowedMentions
}

// BuildWeeklyPayload builds the weekly message. If roleID is not empty, the message
// mentions the role (<@&roleID>) to ping it.
func BuildWeeklyPayload(view appavailability.WeeklyView, roleID string, display guildconfig.Display) (*WeeklyPayload, error) {
	p := &WeeklyPayload{AllowedMentions: &discordgo.MessageAllowedMentions{}}
	if roleID != "" {
		p.AllowedMentions.Roles = []string{roleID}
	}

	if display != guildconfig.DisplayImage {
		p.Embeds = []*discordgo.MessageEmbed{buildClassicEmbed(view)}
		p.Components = buildClassicComponents()
		if roleID != "" {
			p.Content = "<@&" + roleID + ">"
		}
		return p, nil
	}

	img, err := gridimage.Render(view)
	if err != nil {
		return nil, fmt.Errorf("render grid image: %w", err)
	}
	p.Files = []*discordgo.File{{Name: weeklyImageName, ContentType: "image/webp", Reader: bytes.NewReader(img)}}
	p.Flags = discordgo.MessageFlagsIsComponentsV2
	if roleID != "" {
		p.Components = append(p.Components, discordgo.TextDisplay{Content: "<@&" + roleID + ">"})
	}
	p.Components = append(p.Components, buildImageContainer(view))
	return p, nil
}

// MessageSend converts the payload into a channel message.
func (p *WeeklyPayload) MessageSend() *discordgo.MessageSend {
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
func (p *WeeklyPayload) MessageEdit(channelID, messageID string) *discordgo.MessageEdit {
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
func buildClassicEmbed(view appavailability.WeeklyView) *discordgo.MessageEmbed {
	description := view.Table
	if view.Legend != "" {
		description = "*" + view.Legend + "*\n" + view.Table
	}
	return &discordgo.MessageEmbed{
		Title:       "Disponibilités de la semaine",
		Description: description,
		Color:       weeklyAccentColor,
	}
}

// buildClassicComponents returns the action row with the original weekly buttons.
func buildClassicComponents() []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{
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
		discordgo.Button{
			Label:    "🚫 Pas dispo cette semaine",
			Style:    discordgo.SecondaryButton,
			CustomID: UnavailableCustomIDPrefix,
		},
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

// buildImageContainer lays out the "image" display: title, grid image, actions.
func buildImageContainer(view appavailability.WeeklyView) discordgo.Container {
	legend := "🟩 beaucoup · 🟨 moyen · ⬛ peu"
	if view.Legend != "" {
		legend = view.Legend + " · " + legend
	}
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "✏️ Mes dispos",
			Style:    discordgo.PrimaryButton,
			CustomID: OpenDisposCustomIDPrefix,
		},
		discordgo.Button{
			Label:    "🚫 Pas dispo cette semaine",
			Style:    discordgo.SecondaryButton,
			CustomID: UnavailableCustomIDPrefix,
		},
	}

	accent := weeklyAccentColor
	return discordgo.Container{
		AccentColor: &accent,
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "## " + view.Title + "\n-# " + legend},
			discordgo.MediaGallery{
				Items: []discordgo.MediaGalleryItem{{Media: discordgo.UnfurledMediaItem{URL: "attachment://" + weeklyImageName}}},
			},
			discordgo.ActionsRow{Components: buttons},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{buildWhoDisposSelect()},
			},
		},
	}
}

// buildWhoDisposSelect lets members see who is available on a given day (answered in ephemeral).
// Discord has no clickable grid, so this select replaces "clicking a cell".
func buildWhoDisposSelect() discordgo.SelectMenu {
	options := []discordgo.SelectMenuOption{
		{Label: "Toute la semaine", Value: whoDisposAllValue, Emoji: &discordgo.ComponentEmoji{Name: "📋"}},
	}
	for day := 0; day < availability.DaysPerWeek; day++ {
		options = append(options, discordgo.SelectMenuOption{
			Label: appavailability.DayLabel(day),
			Value: strconv.Itoa(day),
		})
	}
	return discordgo.SelectMenu{
		CustomID:    WhoDisposCustomIDPrefix,
		Placeholder: "👀 Voir qui est dispo…",
		Options:     options,
	}
}
