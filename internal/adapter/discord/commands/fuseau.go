package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/i18n"
	"github.com/bwmarrin/discordgo"
)

const CommandNameFuseau = "fuseau"

// FuseauCommand lets members set the timezone used to show them the hourly slots.
type FuseauCommand struct {
	svc      *appavailability.Service
	guildCfg *appguildconfig.Service
}

// NewFuseauCommand creates the /fuseau command.
func NewFuseauCommand(svc *appavailability.Service, guildCfg *appguildconfig.Service) *FuseauCommand {
	return &FuseauCommand{svc: svc, guildCfg: guildCfg}
}

func (c *FuseauCommand) Name() string {
	return CommandNameFuseau
}

func (c *FuseauCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	desc, descEN := description(func(m *i18n.Messages) string { return m.FuseauDescription })
	optDesc, optDescEN := description(func(m *i18n.Messages) string { return m.FuseauOptionDesc })
	return &discordgo.ApplicationCommand{
		Name:                     CommandNameFuseau,
		Description:              desc,
		DescriptionLocalizations: descEN,
		Options: []*discordgo.ApplicationCommandOption{{
			Type:                     discordgo.ApplicationCommandOptionString,
			Name:                     "fuseau",
			Description:              optDesc,
			DescriptionLocalizations: *optDescEN,
			Required:                 true,
			Autocomplete:             true,
		}},
	}
}

var (
	_ discord.Command       = (*FuseauCommand)(nil)
	_ discord.Autocompleter = (*FuseauCommand)(nil)
)

// messages returns the texts in the language of the interaction's guild (French in DMs).
func (c *FuseauCommand) messages(i *discordgo.InteractionCreate) *i18n.Messages {
	cfg, err := c.guildCfg.Get(context.Background(), i.GuildID)
	if err != nil {
		log.Printf("fuseau: get guild config: %v", err)
		return i18n.For(i18n.FR)
	}
	return cfg.Messages()
}

func (c *FuseauCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := c.messages(i)
	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	input := newOptionMap(i.ApplicationCommandData().Options).str("fuseau")

	loc, err := availability.ParseTimezone(input)
	if err != nil || userID == "" {
		return respondWithError(s, i, fmt.Sprintf(m.TimezoneUnknown, input))
	}
	if err := c.svc.SetUserTimezone(context.Background(), userID, loc); err != nil {
		log.Printf("fuseau: set user timezone: %v", err)
		return respondWithError(s, i, m.ErrGeneric)
	}
	return respondEphemeral(s, i, fmt.Sprintf(m.TimezoneSaved, m.TimezoneName(loc), m.Clock(time.Now().In(loc)))+"\n"+m.TimezoneUsedBy)
}

// Autocomplete suggests the countries matching what is typed (in French or English, or
// the IANA name), plus what is typed itself when it is a valid timezone or UTC offset.
func (c *FuseauCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	m := c.messages(i)
	query := strings.TrimSpace(newOptionMap(i.ApplicationCommandData().Options).str("fuseau"))
	now := time.Now()

	var choices []*discordgo.ApplicationCommandOptionChoice
	if loc, err := availability.ParseTimezone(query); err == nil {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s · %s", m.Clock(now.In(loc)), m.TimezoneName(loc)),
			Value: loc.String(),
		})
	}
	needle := strings.ToLower(query)
	for _, tz := range availability.Countries {
		if len(choices) == 25 {
			break
		}
		if needle != "" && !strings.Contains(strings.ToLower(tz.Fr+" "+tz.En+" "+tz.Zone), needle) {
			continue
		}
		if len(choices) > 0 && choices[0].Value == tz.Zone {
			continue
		}
		loc, err := time.LoadLocation(tz.Zone)
		if err != nil {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s · %s", m.Clock(now.In(loc)), m.CountryName(tz)),
			Value: tz.Zone,
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
