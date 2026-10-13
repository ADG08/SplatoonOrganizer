package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

const CommandNameFuseau = "fuseau"

// FuseauCommand lets members set the timezone used to show them the hourly slots.
type FuseauCommand struct {
	svc *appavailability.Service
}

// NewFuseauCommand creates the /fuseau command.
func NewFuseauCommand(svc *appavailability.Service) *FuseauCommand {
	return &FuseauCommand{svc: svc}
}

func (c *FuseauCommand) Name() string {
	return CommandNameFuseau
}

func (c *FuseauCommand) ApplicationCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        CommandNameFuseau,
		Description: "Choisir ton fuseau horaire pour voir les créneaux à ton heure",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "fuseau",
			Description:  "Ta ville ou ton décalage (ex : Montréal, America/Lima, UTC+5:30)",
			Required:     true,
			Autocomplete: true,
		}},
	}
}

var (
	_ discord.Command       = (*FuseauCommand)(nil)
	_ discord.Autocompleter = (*FuseauCommand)(nil)
)

func (c *FuseauCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	input := newOptionMap(i.ApplicationCommandData().Options).str("fuseau")

	loc, err := availability.ParseTimezone(input)
	if err != nil || userID == "" {
		return respondWithError(s, i, fmt.Sprintf("Je ne connais pas le fuseau « %s ». Choisis une proposition de la liste, "+
			"ou écris un décalage comme `UTC+5:30`.", input))
	}
	if err := c.svc.SetUserTimezone(context.Background(), userID, loc); err != nil {
		log.Printf("fuseau: set user timezone: %v", err)
		return respondWithError(s, i, "Erreur lors de l'enregistrement de ton fuseau.")
	}
	return respondEphemeral(s, i, fmt.Sprintf("✅ Fuseau enregistré : **%s** (il est %s chez toi).\n"+
		"Les menus **✏️ Mes dispos** et le bouton **🌍 Mon fuseau** l'utiliseront.",
		availability.TimezoneName(loc), time.Now().In(loc).Format("15:04")))
}

// Autocomplete suggests the common timezones matching what is typed (city, country or
// IANA name), plus what is typed itself when it is a valid timezone or UTC offset.
func (c *FuseauCommand) Autocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	query := strings.TrimSpace(newOptionMap(i.ApplicationCommandData().Options).str("fuseau"))
	now := time.Now()

	var choices []*discordgo.ApplicationCommandOptionChoice
	if loc, err := availability.ParseTimezone(query); err == nil {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%s · %s", now.In(loc).Format("15:04"), availability.TimezoneName(loc)),
			Value: loc.String(),
		})
	}
	needle := strings.ToLower(query)
	for _, tz := range availability.Countries {
		if len(choices) == 25 {
			break
		}
		if needle != "" && !strings.Contains(strings.ToLower(tz.Where+" "+tz.Zone), needle) {
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
			Name:  fmt.Sprintf("%s · %s", now.In(loc).Format("15:04"), tz.Where),
			Value: tz.Zone,
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
