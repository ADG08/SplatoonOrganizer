package weekly

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/team"
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
	p, err := BuildPayload(&team.Team{ID: 7, Name: "Team A", RoleID: "r1", Display: team.DisplayClassic, Language: i18n.FR}, view(t))
	if err != nil {
		t.Fatal(err)
	}
	if p.Content != "<@&r1>" || len(p.AllowedMentions.Roles) != 1 || p.AllowedMentions.Roles[0] != "r1" {
		t.Errorf("role ping: content %q, allowed %v", p.Content, p.AllowedMentions.Roles)
	}
	if len(p.Embeds) != 1 || p.Embeds[0].Title != "Disponibilités de la semaine · Team A" || p.Flags != 0 {
		t.Errorf("classic embed: %+v, flags %v", p.Embeds, p.Flags)
	}
	buttons := p.Components[0].(discordgo.ActionsRow).Components
	var ids []string
	for _, b := range buttons {
		ids = append(ids, b.(discordgo.Button).CustomID)
	}
	want := []string{"availability:open:7", "availability:view:7", "availability:unavailable:7"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("buttons: %v, want %v", ids, want)
	}
}

func TestImagePayload(t *testing.T) {
	p, err := BuildPayload(&team.Team{Name: "Team A", Display: team.DisplayImage, Language: i18n.EN}, view(t))
	if err != nil {
		t.Fatal(err)
	}
	if p.Flags != discordgo.MessageFlagsIsComponentsV2 || p.Content != "" || len(p.Embeds) != 0 {
		t.Errorf("Components V2 messages have no content nor embeds: %+v", p)
	}
	if len(p.Files) != 1 || p.Files[0].Name != ImageName || len(p.AllowedMentions.Roles) != 0 {
		t.Errorf("files %v, allowed roles %v", p.Files, p.AllowedMentions.Roles)
	}
	container, ok := p.Components[0].(discordgo.Container)
	if !ok || len(p.Components) != 1 {
		t.Fatalf("components: %T", p.Components)
	}
	if title := container.Components[0].(discordgo.TextDisplay).Content; !strings.Contains(title, " · Team A") {
		t.Errorf("the title doesn't name the team: %q", title)
	}

	// Editing replaces the previous image: only the new upload is kept, by index.
	edit := p.MessageEdit("c1", "m1")
	if edit.Attachments == nil || len(*edit.Attachments) != 1 || (*edit.Attachments)[0].ID != "0" {
		t.Errorf("attachments: %v", edit.Attachments)
	}
}

// Two teams of a guild can be set up differently: each message follows its own team.
func TestPayloadsFollowEachTeam(t *testing.T) {
	classicFR, err := BuildPayload(&team.Team{ID: 1, Name: "A", Language: i18n.FR}, view(t))
	if err != nil {
		t.Fatal(err)
	}
	imageEN, err := BuildPayload(&team.Team{ID: 2, Name: "B", Language: i18n.EN, Display: team.DisplayImage, FullAt: 4}, view(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(classicFR.Embeds) != 1 || len(imageEN.Embeds) != 0 {
		t.Errorf("display: %d and %d embeds", len(classicFR.Embeds), len(imageEN.Embeds))
	}
	raw, _ := json.Marshal(imageEN.Components)
	if !strings.Contains(string(raw), "🟩 4+") || !strings.Contains(string(raw), i18n.For(i18n.EN).BtnMyDispos) {
		t.Errorf("the second team's threshold or language is missing: %s", raw)
	}
}

func TestParseCustomID(t *testing.T) {
	for _, c := range []struct {
		customID, prefix string
		teamID           int64
		rest             string
		wantErr          bool
	}{
		{CustomID(OpenDisposCustomID, 12), OpenDisposCustomID, 12, "", false},
		{CustomID("availability:select", 3, "6"), "availability:select", 3, "6", false},
		{CustomID("availability:tz", 3, "dispos", "w"), "availability:tz", 3, "dispos:w", false},
		{OpenDisposCustomID, OpenDisposCustomID, 0, "", false}, // posted before teams existed
		{OpenDisposCustomID + ":abc", OpenDisposCustomID, 0, "", true},
		{OpenDisposCustomID + ":-1", OpenDisposCustomID, 0, "", true},
		{OpenDisposCustomID + ":0", OpenDisposCustomID, 0, "", true},
		{ViewDisposCustomID + ":1", OpenDisposCustomID, 0, "", true},
		{"availability:tzedit:1", "availability:tz", 0, "", true},
	} {
		teamID, rest, err := ParseCustomID(c.customID, c.prefix)
		if (err != nil) != c.wantErr {
			t.Errorf("%q: error %v, want error %v", c.customID, err, c.wantErr)
			continue
		}
		if teamID != c.teamID || rest != c.rest {
			t.Errorf("%q: got %d, %q, want %d, %q", c.customID, teamID, rest, c.teamID, c.rest)
		}
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
	hasMyTime := func(cfg *team.Team) bool {
		p, err := BuildPayload(cfg, view(t))
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(p.Components)
		return strings.Contains(string(raw), MyTimeCustomID)
	}
	for _, c := range []struct {
		cfg  team.Team
		want bool
	}{
		{team.Team{SlotLayout: hourly, MemberTimezones: true}, true},
		{team.Team{SlotLayout: hourly, MemberTimezones: true, Display: team.DisplayImage}, true},
		{team.Team{SlotLayout: hourly}, false},                                              // everyone on the team's time
		{team.Team{SlotLayout: availability.HalfDayLayout(), MemberTimezones: true}, false}, // no hours
	} {
		if got := hasMyTime(&c.cfg); got != c.want {
			t.Errorf("%+v: Mon fuseau shown %v, want %v", c.cfg, got, c.want)
		}
	}
}
