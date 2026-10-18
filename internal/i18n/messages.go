package i18n

// Messages holds the texts of one language. Fields ending with a format verb are used
// with fmt.Sprintf; a test checks both languages fill every field with the same verbs.
type Messages struct {
	lang Lang

	// Calendar and slots
	Days          [7]string // Monday first
	ShortDays     [7]string // 3 letters: they align the text tables
	HalfDay       [2]string // the two slots of the half-day layout
	Midnight      string
	HourlySummary string // %s start, %s end
	DateFormat    string // Go layout of the week's Monday in titles

	// Weekly message
	WeeklyTitle      string // %s Monday's date
	ClassicTitle     string
	TableDayHeader   string // first column of the half-day table, 10 characters at most
	ServerTimeLegend string // %s server timezone, %s an hour there, %s same hour for the member
	ImageLegend      string
	ClosedLegend     string
	BtnMyDispos      string
	BtnViewDispos    string
	BtnUnavailable   string
	BtnMyTimezone    string
	WhoPlaceholder   string
	WhoWholeWeek     string
	WhoTitle         string // %s day
	ViewTitle        string
	YourTimezoneNote string
	NobodyThatDay    string
	UnavailableUsers string // %s mentions

	// "Mes dispos" and "Pas dispo cette semaine"
	DisposIntro     string
	DisposMore      string
	DisposUpdated   string
	DayClosed       string // %s day
	AvailableAgain  string
	UnavailableNow  string
	TimezoneInMenus string // %s timezone name

	// Timezones
	PickerContent      string
	PickerWest         string
	PickerEast         string
	PickerCountry      string // %s UTC offset
	CountryPlaceholder string
	OtherCountry       string // %s UTC offset
	OtherCountryHint   string
	BtnEditTimezone    string
	TimezoneSaved      string // %s timezone name, %s current time there
	GridInYourTime     string // %s weekly title
	LocalTimeLegend    string // %s timezone name
	ShiftedColumns     string // %d minutes
	NoHoursToConvert   string
	HelpDescription    string
	AnnonceDescription string
	AnnonceTitleDesc   string
	AnnonceMessageDesc string
	AnnonceRoleDesc    string

	// /help panel: three sections, each with its button and form
	PanelTitle       string
	PanelDescription string
	PanelFooter      string
	SectionPublish   string
	SectionSlots     string
	SectionTimezone  string
	SectionMessage   string
	FieldChannel     string
	FieldRole        string
	FieldWhen        string
	FieldDisplay     string
	FieldLanguage    string
	NotConfigured    string
	NoRole           string
	PausedTitle      string
	PausedHint       string
	BtnEdit          string
	BtnSupport       string
	ConfigSaved      string
	NothingChanged   string
	PublishedNow     string

	// /help: publication form
	ChannelLabel     string
	ChannelHint      string
	RoleLabel        string
	RoleHint         string
	PublishDayLabel  string
	PublishDayHint   string
	PublishSunday    string
	PublishMonday    string
	PublishHourLabel string
	PublishHourHint  string

	// /help: slots form
	ModeLabel        string
	ModeHint         string
	ModeHourly       string
	StartHourLabel   string
	StartHourHint    string
	EndHourLabel     string
	EndHourHint      string
	NextDay          string // after an hour of the next morning
	HourOrder        string
	MemberTimeLabel  string
	MemberTimeHint   string
	MemberTimeServer string
	MemberTimeOwn    string
	FieldMembers     string
	MembersServer    string
	MembersOwn       string
	MemberTimeOff    string
	SlotsReset       string

	// /help: message form
	LanguageLabel  string
	LanguageNames  [2]string // in the order of Langs
	DisplayLabel   string
	DisplayHint    string
	DisplayClassic string
	DisplayImage   string
	FullAtLabel    string
	FullAtHint     string
	FullAtAuto     string
	FieldFullAt    string
	PlayersOne     string
	PlayersMany    string // %d players

	// /help: timezone of the guild
	GuildPickerContent string // %s current timezone
	GuildPickerCountry string // %s UTC offset
	GuildTimezoneSaved string // %s timezone name, %s current time there

	// /annonce
	AnnounceBy        string // %s mention
	AnnouncePublished string // %s channel mention
	AnnounceEmpty     string
	NoChannel         string

	// Errors
	ErrGeneric     string
	ErrUnknownUser string
	ErrInvalid     string
	ErrAdminOnly   string
	ErrGuildOnly   string
}

var fr = Messages{
	lang: FR,

	Days:          [7]string{"Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi", "Dimanche"},
	ShortDays:     [7]string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"},
	HalfDay:       [2]string{"Après-midi", "Soir"},
	Midnight:      "minuit",
	HourlySummary: "Par heure · %s → %s",
	DateFormat:    "02/01",

	WeeklyTitle:      "Dispos de la semaine du %s",
	ClassicTitle:     "Disponibilités de la semaine",
	TableDayHeader:   "Jour",
	ServerTimeLegend: "Heure du serveur (%s) · %s = %s chez toi",
	ImageLegend:      "🟩 beaucoup · 🟨 moyen · ⬛ peu",
	ClosedLegend:     "cases barrées : avant la publication",
	BtnMyDispos:      "✏️ Mes dispos",
	BtnViewDispos:    "📋 Voir les dispos",
	BtnUnavailable:   "🚫 Pas dispo cette semaine",
	BtnMyTimezone:    "🌍 Mon fuseau",
	WhoPlaceholder:   "👀 Voir qui est dispo…",
	WhoWholeWeek:     "Toute la semaine",
	WhoTitle:         "Qui est dispo %s ?",
	ViewTitle:        "Disponibilités par créneau",
	YourTimezoneNote: "🕒 Heures affichées dans ton fuseau horaire",
	NobodyThatDay:    "Personne n'est dispo ce jour-là pour l'instant.",
	UnavailableUsers: "🚫 **Indisponibles cette semaine** : %s",

	DisposIntro:     "Configure tes disponibilités de la semaine :",
	DisposMore:      "Suite :",
	DisposUpdated:   "Tes disponibilités ont été mises à jour.",
	DayClosed:       "%s · avant la publication",
	AvailableAgain:  "✅ Tu es de nouveau disponible cette semaine. Configure tes créneaux avec **✏️ Mes dispos**.",
	UnavailableNow:  "🚫 Tu es marqué **indisponible** pour toute la semaine. Tes créneaux ont été effacés. Reclique sur le bouton (ou coche un créneau) pour revenir disponible.",
	TimezoneInMenus: "-# 🌍 Heures affichées à ton heure (%s)",

	PickerContent: "🌍 **Quel est ton fuseau horaire ?**\n" +
		"Choisis ton décalage UTC (l'heure affichée doit être ton heure actuelle), puis ton pays : " +
		"les créneaux s'afficheront ensuite à ton heure, changements d'heure compris.",
	PickerWest:         "UTC-12 → UTC-1 · Amériques, Pacifique",
	PickerEast:         "UTC+0 → UTC+14 · Europe, Afrique, Asie, Océanie",
	PickerCountry:      "🌍 **Dans quel pays es-tu ?** (%s)\nLe bot suivra tout seul les passages à l'heure d'été et d'hiver.",
	CountryPlaceholder: "Ton pays…",
	OtherCountry:       "Autre pays (%s fixe)",
	OtherCountryHint:   "Ne suit pas les changements d'heure",
	BtnEditTimezone:    "⚙️ Modifier mon fuseau",
	TimezoneSaved:      "✅ Fuseau enregistré : **%s** (il est %s chez toi).",
	GridInYourTime:     "%s · à ton heure",
	LocalTimeLegend:    "Heures affichées à ton heure · %s",
	ShiftedColumns:     " · colonnes décalées de %d min",
	NoHoursToConvert:   "Les créneaux de ce serveur sont « Après-midi / Soir » : il n'y a pas d'heure à convertir.",
	HelpDescription:    "Panneau admin pour configurer le bot",
	AnnonceDescription: "Publier une annonce (embed) dans le salon des dispos",
	AnnonceTitleDesc:   "Titre de l'annonce",
	AnnonceMessageDesc: "Contenu de l'annonce (utilise \\n pour un retour à la ligne)",
	AnnonceRoleDesc:    "Rôle à mentionner (optionnel)",

	PanelTitle:       "⚙️ Panneau de configuration",
	PanelDescription: "Gère les paramètres du bot pour ce serveur.",
	PanelFooter:      "🔒 Réservé aux administrateurs",
	SectionPublish:   "📢 Publication",
	SectionSlots:     "🕒 Créneaux",
	SectionTimezone:  "🌍 Fuseau horaire",
	SectionMessage:   "🎨 Message",
	FieldChannel:     "Salon",
	FieldRole:        "Rôle",
	FieldWhen:        "Quand",
	FieldDisplay:     "Affichage",
	FieldLanguage:    "Langue",
	NotConfigured:    "❌ non configuré",
	NoRole:           "aucun",
	PausedTitle:      "⏸️ Publication en pause",
	PausedHint:       "Le bot ne peut plus publier dans ce salon. Vérifie ses permissions, puis choisis à nouveau le salon.",
	BtnEdit:          "✏️ Modifier",
	BtnSupport:       "💸 Soutenir le bot",
	ConfigSaved:      "Configuration enregistrée.",
	NothingChanged:   "Rien n'a changé.",
	PublishedNow:     " Le message de la semaine est en cours de publication.",

	ChannelLabel:     "Salon des dispos",
	ChannelHint:      "Où le message de la semaine est publié.",
	RoleLabel:        "Rôle à mentionner",
	RoleHint:         "Optionnel : sans rôle, personne n'est mentionné.",
	PublishDayLabel:  "Jour de publication",
	PublishDayHint:   "Le message couvre toujours une semaine du lundi au dimanche.",
	PublishSunday:    "Dimanche, pour la semaine qui suit",
	PublishMonday:    "Lundi, pour la semaine qui commence",
	PublishHourLabel: "Heure de publication",
	PublishHourHint:  "Heure du serveur.",

	ModeLabel:        "Découpage des journées",
	ModeHint:         "⚠️ Changer le découpage réinitialise les dispos de la semaine en cours.",
	ModeHourly:       "Par heure",
	StartHourLabel:   "Première heure (mode par heure)",
	StartHourHint:    "Heure du serveur. Ignorée en Après-midi / Soir.",
	EndHourLabel:     "Fin du dernier créneau (mode par heure)",
	EndHourHint:      "Jusqu'à 5h le lendemain. Ignorée en Après-midi / Soir.",
	NextDay:          " (lendemain)",
	HourOrder:        "La première heure doit être avant la fin du dernier créneau.",
	MemberTimeLabel:  "Heures des membres (mode par heure)",
	MemberTimeHint:   "International : chacun indique son fuseau et voit les créneaux à son heure.",
	MemberTimeServer: "🏠 Tout le monde à l'heure du serveur",
	MemberTimeOwn:    "🌍 International : chacun à son heure",
	FieldMembers:     "Membres",
	MembersServer:    "à l'heure du serveur",
	MembersOwn:       "chacun à son heure",
	MemberTimeOff:    "Sur ce serveur, tout le monde voit les créneaux à l'heure du serveur.",
	SlotsReset:       " Les créneaux ayant changé, les dispos de la semaine ont été réinitialisées.",

	LanguageLabel:  "Langue du bot",
	LanguageNames:  [2]string{"Français", "English"},
	DisplayLabel:   "Affichage du message",
	DisplayHint:    "Changer l'affichage ou la langue republie le message de la semaine.",
	DisplayClassic: "Classique (texte)",
	DisplayImage:   "Image",
	FullAtLabel:    "Case verte à partir de (affichage image)",
	FullAtHint:     "Jaune dès la moitié, gris en dessous. Automatique : selon le meilleur créneau de la semaine.",
	FullAtAuto:     "Automatique",
	FieldFullAt:    "Vert dès",
	PlayersOne:     "1 joueur",
	PlayersMany:    "%d joueurs",

	GuildPickerContent: "🌍 **Quel est le fuseau horaire du serveur ?** (actuellement : %s)\nToutes ses heures le suivent : créneaux, grille, semaines et publication. Choisis le décalage UTC actuel, puis le pays.",
	GuildPickerCountry: "🌍 **Dans quel pays est le serveur ?** (%s)\nLe bot suivra tout seul les passages à l'heure d'été et d'hiver.",
	GuildTimezoneSaved: "Fuseau du serveur : **%s** (il y est %s).",

	AnnounceBy:        "**Annonce de** %s",
	AnnouncePublished: "✅ Annonce publiée dans %s.",
	AnnounceEmpty:     "Le titre et le message ne peuvent pas être vides.",
	NoChannel:         "Aucun salon configuré. Utilise `/help` pour définir le salon des dispos.",

	ErrGeneric:     "Une erreur est survenue, réessaie dans un instant.",
	ErrUnknownUser: "Impossible de déterminer ton utilisateur Discord pour cette interaction.",
	ErrInvalid:     "Interaction invalide.",
	ErrAdminOnly:   "Tu dois être administrateur du serveur ou le créateur du bot pour utiliser ceci.",
	ErrGuildOnly:   "Cette commande doit être utilisée sur un serveur.",
}

var en = Messages{
	lang: EN,

	Days:          [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
	ShortDays:     [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
	HalfDay:       [2]string{"Afternoon", "Evening"},
	Midnight:      "midnight",
	HourlySummary: "Hourly · %s → %s",
	DateFormat:    "Jan 2",

	WeeklyTitle:      "Availability for the week of %s",
	ClassicTitle:     "This week's availability",
	TableDayHeader:   "Day",
	ServerTimeLegend: "Server time (%s) · %s = %s for you",
	ImageLegend:      "🟩 many · 🟨 some · ⬛ few",
	ClosedLegend:     "crossed-out cells: before posting",
	BtnMyDispos:      "✏️ My availability",
	BtnViewDispos:    "📋 See availability",
	BtnUnavailable:   "🚫 Not available this week",
	BtnMyTimezone:    "🌍 My timezone",
	WhoPlaceholder:   "👀 See who's available…",
	WhoWholeWeek:     "Whole week",
	WhoTitle:         "Who's available on %s?",
	ViewTitle:        "Availability by slot",
	YourTimezoneNote: "🕒 Hours shown in your timezone",
	NobodyThatDay:    "Nobody is available that day yet.",
	UnavailableUsers: "🚫 **Unavailable this week**: %s",

	DisposIntro:     "Set your availability for the week:",
	DisposMore:      "More:",
	DisposUpdated:   "Your availability has been updated.",
	DayClosed:       "%s · before posting",
	AvailableAgain:  "✅ You're available again this week. Pick your slots with **✏️ My availability**.",
	UnavailableNow:  "🚫 You're marked **unavailable** for the whole week. Your slots were cleared. Click the button again (or pick a slot) to be available again.",
	TimezoneInMenus: "-# 🌍 Hours shown in your time (%s)",

	PickerContent: "🌍 **What's your timezone?**\n" +
		"Pick your UTC offset (the time shown should be your current time), then your country: " +
		"slots will then be shown in your time, clock changes included.",
	PickerWest:         "UTC-12 → UTC-1 · Americas, Pacific",
	PickerEast:         "UTC+0 → UTC+14 · Europe, Africa, Asia, Oceania",
	PickerCountry:      "🌍 **Which country are you in?** (%s)\nThe bot will follow daylight saving time changes by itself.",
	CountryPlaceholder: "Your country…",
	OtherCountry:       "Another country (fixed %s)",
	OtherCountryHint:   "Doesn't follow clock changes",
	BtnEditTimezone:    "⚙️ Change my timezone",
	TimezoneSaved:      "✅ Timezone saved: **%s** (it is %s for you).",
	GridInYourTime:     "%s · in your time",
	LocalTimeLegend:    "Hours shown in your time · %s",
	ShiftedColumns:     " · columns shifted by %d min",
	NoHoursToConvert:   "This server uses \"Afternoon / Evening\" slots: there are no hours to convert.",
	HelpDescription:    "Admin panel to configure the bot",
	AnnonceDescription: "Post an announcement (embed) in the availability channel",
	AnnonceTitleDesc:   "Title of the announcement",
	AnnonceMessageDesc: "Content of the announcement (use \\n for a line break)",
	AnnonceRoleDesc:    "Role to mention (optional)",

	PanelTitle:       "⚙️ Settings",
	PanelDescription: "Manage the bot's settings for this server.",
	PanelFooter:      "🔒 Admins only",
	SectionPublish:   "📢 Posting",
	SectionSlots:     "🕒 Slots",
	SectionTimezone:  "🌍 Timezone",
	SectionMessage:   "🎨 Message",
	FieldChannel:     "Channel",
	FieldRole:        "Role",
	FieldWhen:        "When",
	FieldDisplay:     "Display",
	FieldLanguage:    "Language",
	NotConfigured:    "❌ not set",
	NoRole:           "none",
	PausedTitle:      "⏸️ Posting paused",
	PausedHint:       "The bot can no longer post in this channel. Check its permissions, then pick the channel again.",
	BtnEdit:          "✏️ Edit",
	BtnSupport:       "💸 Support the bot",
	ConfigSaved:      "Settings saved.",
	NothingChanged:   "Nothing changed.",
	PublishedNow:     " The weekly message is being posted.",

	ChannelLabel:     "Availability channel",
	ChannelHint:      "Where the weekly message is posted.",
	RoleLabel:        "Role to mention",
	RoleHint:         "Optional: without a role, nobody is mentioned.",
	PublishDayLabel:  "Posting day",
	PublishDayHint:   "The message always covers a week from Monday to Sunday.",
	PublishSunday:    "Sunday, for the following week",
	PublishMonday:    "Monday, for the week starting",
	PublishHourLabel: "Posting time",
	PublishHourHint:  "Server time.",

	ModeLabel:        "Day slots",
	ModeHint:         "⚠️ Changing the slots resets this week's availability.",
	ModeHourly:       "Hourly",
	StartHourLabel:   "First hour (hourly slots)",
	StartHourHint:    "Server time. Ignored for Afternoon / Evening.",
	EndHourLabel:     "End of the last slot (hourly slots)",
	EndHourHint:      "Up to 5 AM the next day. Ignored for Afternoon / Evening.",
	NextDay:          " (next day)",
	HourOrder:        "The first hour must be before the end of the last slot.",
	MemberTimeLabel:  "Members' hours (hourly slots)",
	MemberTimeHint:   "International: everyone sets their timezone and sees the slots in their own time.",
	MemberTimeServer: "🏠 Everyone on the server's time",
	MemberTimeOwn:    "🌍 International: everyone in their own time",
	FieldMembers:     "Members",
	MembersServer:    "on the server's time",
	MembersOwn:       "each in their own time",
	MemberTimeOff:    "On this server, everyone sees the slots in the server's time.",
	SlotsReset:       " The slots changed, so this week's availability was reset.",

	LanguageLabel:  "Bot language",
	LanguageNames:  [2]string{"Français", "English"},
	DisplayLabel:   "Message display",
	DisplayHint:    "Changing the display or the language posts the weekly message again.",
	DisplayClassic: "Classic (text)",
	DisplayImage:   "Image",
	FullAtLabel:    "Green cell from (image display)",
	FullAtHint:     "Yellow from half of it, grey below. Automatic: against the week's best slot.",
	FullAtAuto:     "Automatic",
	FieldFullAt:    "Green from",
	PlayersOne:     "1 player",
	PlayersMany:    "%d players",

	GuildPickerContent: "🌍 **What's the server's timezone?** (currently: %s)\nAll its hours follow it: slots, grid, weeks and posting. Pick the current UTC offset, then the country.",
	GuildPickerCountry: "🌍 **Which country is the server in?** (%s)\nThe bot will follow daylight saving time changes by itself.",
	GuildTimezoneSaved: "Server timezone: **%s** (it is %s there).",

	AnnounceBy:        "**Announcement by** %s",
	AnnouncePublished: "✅ Announcement posted in %s.",
	AnnounceEmpty:     "The title and the message can't be empty.",
	NoChannel:         "No channel set. Use `/help` to set the availability channel.",

	ErrGeneric:     "Something went wrong, please try again in a moment.",
	ErrUnknownUser: "Couldn't determine your Discord user for this interaction.",
	ErrInvalid:     "Invalid interaction.",
	ErrAdminOnly:   "You must be a server admin or the bot's creator to use this.",
	ErrGuildOnly:   "This command must be used in a server.",
}
