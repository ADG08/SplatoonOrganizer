package weekly

import (
	"testing"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

func view(t *testing.T) appavailability.WeeklyView {
	t.Helper()
	svc := appavailability.NewService(nil, nil)
	counts := []availability.SlotCount{{DayIndex: 0, SlotIndex: 1, Count: 3}}
	return svc.BuildWeeklyView(i18n.For(i18n.FR), availability.HalfDayLayout(), "2026-W42", counts)
}

func TestClassicPayload(t *testing.T) {
	p, err := BuildPayload(i18n.For(i18n.FR), view(t), "r1", guildconfig.DisplayClassic)
	if err != nil {
		t.Fatal(err)
	}
	if p.Content != "<@&r1>" || len(p.AllowedMentions.Roles) != 1 || p.AllowedMentions.Roles[0] != "r1" {
		t.Errorf("role ping: content %q, allowed %v", p.Content, p.AllowedMentions.Roles)
	}
	// The embed and buttons of messages posted before the image display must not change.
	if len(p.Embeds) != 1 || p.Embeds[0].Title != "Disponibilités de la semaine" || p.Flags != 0 {
		t.Errorf("classic embed: %+v, flags %v", p.Embeds, p.Flags)
	}
	buttons := p.Components[0].(discordgo.ActionsRow).Components
	var ids []string
	for _, b := range buttons {
		ids = append(ids, b.(discordgo.Button).CustomID)
	}
	if want := []string{OpenDisposCustomID, ViewDisposCustomID, UnavailableCustomID}; len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] || ids[2] != want[2] {
		t.Errorf("buttons: %v, want %v", ids, want)
	}
}

func TestImagePayload(t *testing.T) {
	p, err := BuildPayload(i18n.For(i18n.EN), view(t), "", guildconfig.DisplayImage)
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags != discordgo.MessageFlagsIsComponentsV2 || p.Content != "" || len(p.Embeds) != 0 {
		t.Errorf("Components V2 messages have no content nor embeds: %+v", p)
	}
	if len(p.Files) != 1 || p.Files[0].Name != ImageName || len(p.AllowedMentions.Roles) != 0 {
		t.Errorf("files %v, allowed roles %v", p.Files, p.AllowedMentions.Roles)
	}
	if _, ok := p.Components[0].(discordgo.Container); !ok || len(p.Components) != 1 {
		t.Errorf("components: %T", p.Components)
	}

	// Editing replaces the previous image: only the new upload is kept, by index.
	edit := p.MessageEdit("c1", "m1")
	if edit.Attachments == nil || len(*edit.Attachments) != 1 || (*edit.Attachments)[0].ID != "0" {
		t.Errorf("attachments: %v", edit.Attachments)
	}
}
