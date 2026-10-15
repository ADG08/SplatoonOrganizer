package weekly

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

func view(t *testing.T) appavailability.WeeklyView {
	t.Helper()
	svc := appavailability.NewService(nil)
	counts := []availability.SlotCount{{DayIndex: 0, SlotIndex: 1, Count: 3}}
	return svc.BuildWeeklyView(i18n.For(i18n.FR), availability.HalfDayLayout(), availability.DefaultLocation(), availability.DefaultSchedule(), "2026-W42", counts)
}

func TestClassicPayload(t *testing.T) {
	p, err := BuildPayload(&guildconfig.GuildConfig{RoleID: "r1", Display: guildconfig.DisplayClassic, Language: i18n.FR}, view(t))
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
	p, err := BuildPayload(&guildconfig.GuildConfig{Display: guildconfig.DisplayImage, Language: i18n.EN}, view(t))
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

func TestLostAccess(t *testing.T) {
	restErr := func(code int) error {
		return fmt.Errorf("send weekly message: %w", &discordgo.RESTError{Message: &discordgo.APIErrorMessage{Code: code}})
	}
	for code, want := range map[int]bool{
		discordgo.ErrCodeUnknownChannel:     true,
		discordgo.ErrCodeMissingAccess:      true,
		discordgo.ErrCodeMissingPermissions: true,
		discordgo.ErrCodeUnknownMessage:     false, // the previous message was deleted: harmless
	} {
		if got := LostAccess(restErr(code)); got != want {
			t.Errorf("code %d: got %v, want %v", code, got, want)
		}
	}
	if LostAccess(errors.New("timeout")) || LostAccess(nil) {
		t.Error("errors other than Discord's are not a lost access")
	}
}

// "Mon fuseau" only shows when members see the slots in their own timezone.
func TestMyTimeButton(t *testing.T) {
	hourly, _ := availability.HourlyLayout(20, 24)
	hasMyTime := func(cfg *guildconfig.GuildConfig) bool {
		p, err := BuildPayload(cfg, view(t))
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(p.Components)
		return strings.Contains(string(raw), MyTimeCustomID)
	}
	for _, c := range []struct {
		cfg  guildconfig.GuildConfig
		want bool
	}{
		{guildconfig.GuildConfig{SlotLayout: hourly, MemberTimezones: true}, true},
		{guildconfig.GuildConfig{SlotLayout: hourly, MemberTimezones: true, Display: guildconfig.DisplayImage}, true},
		{guildconfig.GuildConfig{SlotLayout: hourly}, false},                                              // everyone on the server's time
		{guildconfig.GuildConfig{SlotLayout: availability.HalfDayLayout(), MemberTimezones: true}, false}, // no hours
	} {
		if got := hasMyTime(&c.cfg); got != c.want {
			t.Errorf("%+v: Mon fuseau shown %v, want %v", c.cfg, got, c.want)
		}
	}
}
