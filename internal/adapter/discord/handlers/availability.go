package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	"github.com/ADG08/SplatoonOrganizer/internal/domain/availability"
	"github.com/bwmarrin/discordgo"
)

const (
	OpenDisposCustomIDPrefix   = "availability:open"
	SelectDisposCustomIDPrefix = "availability_select"
)

// OpenDisposHandler handles the "Mes dispos" button click.
type OpenDisposHandler struct {
	svc *appavailability.Service
}

// NewOpenDisposHandler creates a new open dispos handler.
func NewOpenDisposHandler(svc *appavailability.Service) *OpenDisposHandler {
	return &OpenDisposHandler{svc: svc}
}

func (h *OpenDisposHandler) CustomID() string {
	return OpenDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*OpenDisposHandler)(nil)

func (h *OpenDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		log.Printf("open dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	ctx := context.Background()

	_, userSlots, err := h.svc.GetUserAvailability(ctx, userID)
	if err != nil {
		log.Printf("error getting user availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la récupération de vos disponibilités.")
	}

	// Discord: max 5 Action Rows per message, 1 select menu per row → 5 select menus per message.
	// We have 7 days: first message 5 menus (Lundi–Vendredi), follow-up 2 menus (Samedi–Dimanche).
	componentsFirst := buildUserDisposSelectMenus(userSlots, 0, 5)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Configure tes disponibilités pour la semaine en cours :",
			Flags:      discordgo.MessageFlagsEphemeral,
			Components: componentsFirst,
		},
	}); err != nil {
		return err
	}
	componentsSecond := buildUserDisposSelectMenus(userSlots, 5, 2)
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content:    "Suite :",
		Flags:      discordgo.MessageFlagsEphemeral,
		Components: componentsSecond,
	})
	return err
}

// BuildWeeklyEmbed builds the main embed displaying availability summary.
func BuildWeeklyEmbed(table string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Disponibilités de la semaine",
		Description: table,
		Color:       0xFEE75C,
	}
}

// BuildWeeklyMessage builds the full message (embed + edit button).
// If roleID is not empty, the message includes a role mention (<@&roleID>) for pinging.
func BuildWeeklyMessage(table string, roleID string) *discordgo.MessageSend {
	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			BuildWeeklyEmbed(table),
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "✏️ Mes dispos",
						Style:    discordgo.PrimaryButton,
						CustomID: OpenDisposCustomIDPrefix,
					},
				},
			},
		},
	}
	if roleID != "" {
		msg.Content = "<@&" + roleID + ">"
	}
	return msg
}

// SelectDisposHandler handles select menu interactions for day availability.
type SelectDisposHandler struct {
	svc            *appavailability.Service
	defaultChannel string
}

// NewSelectDisposHandler creates a new select dispos handler.
func NewSelectDisposHandler(svc *appavailability.Service, defaultChannelID string) *SelectDisposHandler {
	return &SelectDisposHandler{
		svc:            svc,
		defaultChannel: defaultChannelID,
	}
}

func (h *SelectDisposHandler) CustomID() string {
	return SelectDisposCustomIDPrefix
}

var _ discord.InteractionHandler = (*SelectDisposHandler)(nil)

func (h *SelectDisposHandler) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	userID := getUserID(i)
	if userID == "" {
		log.Printf("select dispos handler: could not determine user ID from interaction")
		return respondWithError(s, i, "Impossible de déterminer ton utilisateur Discord pour cette interaction.")
	}
	dayIndex, err := parseSelectDisposCustomID(i.MessageComponentData().CustomID)
	if err != nil {
		log.Printf("invalid select custom id: %v", err)
		return respondWithError(s, i, "Interaction invalide.")
	}

	ctx := context.Background()

	// Values are "1" and "2" for Après-midi and Soir
	values := i.MessageComponentData().Values
	var slots [availability.SlotsPerDay]bool
	for _, v := range values {
		switch v {
		case "1":
			slots[0] = true
		case "2":
			slots[1] = true
		}
	}

	week, err := h.svc.SetDayAvailability(ctx, userID, dayIndex, slots)
	if err != nil {
		log.Printf("error setting day availability: %v", err)
		return respondWithError(s, i, "Erreur lors de la mise à jour de vos disponibilités.")
	}

	_, userSlots, err := h.svc.GetUserAvailability(ctx, userID)
	if err != nil {
		log.Printf("error getting user availability after select: %v", err)
		return respondWithError(s, i, "Erreur lors du rafraîchissement.")
	}

	// Update the ephemeral message: same chunk as the interacted menu (5 first days or 2 last days)
	var components []discordgo.MessageComponent
	if dayIndex < 5 {
		components = buildUserDisposSelectMenus(userSlots, 0, 5)
	} else {
		components = buildUserDisposSelectMenus(userSlots, 5, 2)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Tes disponibilités ont été mises à jour.",
			Components: components,
		},
	}); err != nil {
		log.Printf("error responding to select: %v", err)
	}

	// Refresh main weekly table (best-effort)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic while refreshing main table: %v", r)
			}
		}()

		counts, err := h.svc.GetAvailabilitySummary(ctx, week)
		if err != nil {
			log.Printf("error getting availability summary: %v", err)
			return
		}

		content := h.svc.FormatTable(counts)
		embed := BuildWeeklyEmbed(content)

		msgID, err := h.svc.GetSondageMessageID(ctx, week)
		if err != nil {
			log.Printf("error getting sondage message id: %v", err)
			return
		}

		channelID := h.defaultChannel
		if i.ChannelID != "" {
			channelID = i.ChannelID
		}
		_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      msgID,
			Channel: channelID,
			Embeds: &[]*discordgo.MessageEmbed{
				embed,
			},
			Components: &[]discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "✏️ Mes dispos",
							Style:    discordgo.PrimaryButton,
							CustomID: OpenDisposCustomIDPrefix,
						},
					},
				},
			},
		})
		if err != nil {
			log.Printf("error editing weekly message: %v", err)
		}
	}()

	return nil
}

// buildUserDisposSelectMenus builds one select menu per day for days [dayStart, dayStart+dayCount).
func buildUserDisposSelectMenus(userSlots map[int]map[int]bool, dayStart, dayCount int) []discordgo.MessageComponent {
	dayPlaceholders := []string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"}
	slotOptions := []struct {
		Value string
		Label string
	}{
		{"1", "Après-midi"},
		{"2", "Soir"},
	}

	minVals := 0
	maxVals := 2

	var rows []discordgo.MessageComponent
	for day := dayStart; day < dayStart+dayCount && day < availability.DaysPerWeek; day++ {
		daySlots := userSlots[day]
		options := make([]discordgo.SelectMenuOption, availability.SlotsPerDay)
		for slot := 0; slot < availability.SlotsPerDay; slot++ {
			available := daySlots != nil && daySlots[slot]
			options[slot] = discordgo.SelectMenuOption{
				Label:   slotOptions[slot].Label,
				Value:   slotOptions[slot].Value,
				Default: available,
			}
		}
		menu := discordgo.SelectMenu{
			CustomID:    fmt.Sprintf("%s_%d", SelectDisposCustomIDPrefix, day),
			Placeholder: dayPlaceholders[day],
			MinValues:   &minVals,
			MaxValues:   maxVals,
			Options:     options,
		}
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})
	}
	return rows
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func getUserID(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func parseSelectDisposCustomID(customID string) (int, error) {
	if !strings.HasPrefix(customID, SelectDisposCustomIDPrefix+"_") {
		return 0, fmt.Errorf("invalid custom id: %s", customID)
	}
	s := customID[len(SelectDisposCustomIDPrefix)+1:]
	day, err := strconv.Atoi(s)
	if err != nil || day < 0 || day >= availability.DaysPerWeek {
		return 0, fmt.Errorf("invalid day index: %s", s)
	}
	return day, nil
}
