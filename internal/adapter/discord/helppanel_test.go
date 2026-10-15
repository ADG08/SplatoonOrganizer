package discord

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/guildconfig"
	"github.com/bwmarrin/discordgo"
)

func TestHelpPanel(t *testing.T) {
	cfg := &guildconfig.GuildConfig{
		ChannelID:  "c1",
		SlotLayout: availability.HalfDayLayout(),
		Schedule:   availability.DefaultSchedule(),
		Location:   availability.DefaultLocation(),
		Active:     false, // shows the paused notice too
	}
	panel := BuildHelpPanel(cfg, "Configuration enregistrée.", "https://revolut.me/example")
	raw, err := json.Marshal(panel)
	if err != nil {
		t.Fatal(err)
	}
	// Discord counts every component, nested ones included: at most 40 per message.
	if n := strings.Count(string(raw), `"type":`); n > 40 {
		t.Errorf("%d components, Discord allows 40", n)
	}

	container := panel[0].(discordgo.Container)
	var buttons []string
	var text strings.Builder
	support := ""
	for _, c := range container.Components {
		switch c := c.(type) {
		case discordgo.ActionsRow:
			support = c.Components[0].(discordgo.Button).URL
		case discordgo.TextDisplay:
			text.WriteString(c.Content + "\n")
		case discordgo.Section:
			text.WriteString(c.Components[0].(discordgo.TextDisplay).Content + "\n")
			buttons = append(buttons, c.Accessory.(discordgo.Button).CustomID)
		}
	}
	if support != "https://revolut.me/example" {
		t.Errorf("support link: %q", support)
	}
	for _, c := range BuildHelpPanel(cfg, "", "")[0].(discordgo.Container).Components {
		if _, ok := c.(discordgo.ActionsRow); ok {
			t.Error("no support link configured, yet the panel shows one")
		}
	}

	want := []string{HelpPanelOpenPublishID, HelpPanelOpenSlotsID, HelpPanelOpenTimezoneID, HelpPanelOpenMessageID}
	if strings.Join(buttons, ",") != strings.Join(want, ",") {
		t.Errorf("section buttons: %v, want %v", buttons, want)
	}
	for _, want := range []string{"✅ Configuration enregistrée.", "⚠️ <#c1>", "Publication en pause", "France"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("panel misses %q", want)
		}
	}
}
