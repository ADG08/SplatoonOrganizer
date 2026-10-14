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

	// /help panel
	PanelTitle        string
	PanelDescription  string
	PanelUpdated      string
	PanelChannel      string
	PanelRole         string
	PanelMessage      string
	PanelFooter       string
	NotConfigured     string
	ChannelPaused     string
	BtnConfigChannel  string
	BtnConfigRole     string
	BtnConfigMessage  string
	ChannelModalTitle string
	ChannelLabel      string
	ChannelHint       string
	RoleModalTitle    string
	RoleLabel         string
	RoleHint          string
	MessageModalTitle string
	LanguageLabel     string
	LanguageNames     [2]string // in the order of Langs
	DisplayLabel      string
	DisplayHint       string
	DisplayClassic    string
	DisplayImage      string
	ModeLabel         string
	ModeHint          string
	ModeHourly        string
	StartHourLabel    string
	StartHourHint     string
	EndHourLabel      string
	EndHourHint       string
	ConfigSaved       string
	NothingChanged    string
	Republished       string
	SlotsReset        string
	NotPublished      string
	HourOrder         string
	NothingSelected   string

	// /help: publication schedule
	PanelPublication      string
	BtnConfigPublication  string
	PublicationModalTitle string
	PublishDayLabel       string
	PublishDayHint        string
	PublishSunday         string
	PublishMonday         string
	PublishHourLabel      string
	PublishHourHint       string
	PublicationSaved      string // %s schedule summary
	PublishedNow          string

	// /help: timezone of the guild
	PanelTimezone      string
	BtnConfigTimezone  string
	GuildPickerContent string // %s current timezone
	GuildPickerCountry string // %s UTC offset
	GuildTimezoneSaved string // %s timezone name, %s current time there
	TimezoneSlotsReset string

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

	PanelTitle:        "⚙️  Panneau de configuration",
	PanelDescription:  "Gère les paramètres du bot pour ce serveur.",
	PanelUpdated:      "✅  Configuration mise à jour",
	PanelChannel:      "📢  Salon des dispos",
	PanelRole:         "🏷️  Rôle à mentionner",
	PanelMessage:      "🕒  Message",
	PanelFooter:       "🔒 Réservé aux administrateurs",
	NotConfigured:     "❌  Non configuré",
	ChannelPaused:     "⏸️ Publication en pause : le bot ne peut plus publier dans ce salon. Vérifie ses permissions, puis choisis à nouveau le salon.",
	BtnConfigChannel:  "📢  Configurer le salon",
	BtnConfigRole:     "🏷️  Configurer le rôle",
	BtnConfigMessage:  "🕒  Créneaux, affichage, langue",
	ChannelModalTitle: "Configurer le salon",
	ChannelLabel:      "Salon des dispos",
	ChannelHint:       "Choisis un salon texte",
	RoleModalTitle:    "Configurer le rôle",
	RoleLabel:         "Rôle à mentionner",
	RoleHint:          "Choisis le rôle à mentionner",
	MessageModalTitle: "Créneaux, affichage, langue",
	LanguageLabel:     "Langue du bot",
	LanguageNames:     [2]string{"Français", "English"},
	DisplayLabel:      "Affichage du message",
	DisplayHint:       "Changer l'affichage ou la langue republie le message de la semaine.",
	DisplayClassic:    "Classique (texte)",
	DisplayImage:      "Image",
	ModeLabel:         "Découpage des journées",
	ModeHint:          "⚠️ Changer le découpage réinitialise les dispos de la semaine en cours.",
	ModeHourly:        "Par heure",
	StartHourLabel:    "Première heure (mode par heure)",
	StartHourHint:     "Heure du serveur. Ignorée en Après-midi / Soir.",
	EndHourLabel:      "Fin du dernier créneau (mode par heure)",
	EndHourHint:       "Ignorée en Après-midi / Soir.",
	ConfigSaved:       "Configuration enregistrée.",
	NothingChanged:    "Rien n'a changé.",
	Republished:       "Configuration enregistrée : le message de la semaine a été republié.",
	SlotsReset:        "Créneaux mis à jour : les dispos de la semaine ont été réinitialisées et le message republié.",
	NotPublished:      " (aucun salon configuré : rien n'a été publié)",
	HourOrder:         "La première heure doit être avant la fin du dernier créneau.",
	NothingSelected:   "Rien n'a été sélectionné.",

	PanelPublication:      "📅  Publication",
	BtnConfigPublication:  "📅  Publication",
	PublicationModalTitle: "Publication du message",
	PublishDayLabel:       "Jour de publication",
	PublishDayHint:        "Le message couvre toujours une semaine du lundi au dimanche.",
	PublishSunday:         "Dimanche, pour la semaine qui suit",
	PublishMonday:         "Lundi, pour la semaine qui commence",
	PublishHourLabel:      "Heure de publication",
	PublishHourHint:       "Heure du serveur.",
	PublicationSaved:      "Publication enregistrée : %s.",
	PublishedNow:          " Le message de la semaine est en cours de publication.",

	PanelTimezone:      "🌍  Fuseau horaire",
	BtnConfigTimezone:  "🌍  Fuseau horaire",
	GuildPickerContent: "🌍 **Quel est le fuseau horaire du serveur ?** (actuellement : %s)\nToutes ses heures le suivent : créneaux, grille, semaines et publication. Choisis le décalage UTC actuel, puis le pays.",
	GuildPickerCountry: "🌍 **Dans quel pays est le serveur ?** (%s)\nLe bot suivra tout seul les passages à l'heure d'été et d'hiver.",
	GuildTimezoneSaved: "Fuseau du serveur : **%s** (il y est %s).",
	TimezoneSlotsReset: " Les créneaux ayant changé d'heure, les dispos de la semaine ont été réinitialisées.",

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

	PanelTitle:        "⚙️  Settings",
	PanelDescription:  "Manage the bot's settings for this server.",
	PanelUpdated:      "✅  Settings updated",
	PanelChannel:      "📢  Availability channel",
	PanelRole:         "🏷️  Role to mention",
	PanelMessage:      "🕒  Message",
	PanelFooter:       "🔒 Admins only",
	NotConfigured:     "❌  Not set",
	ChannelPaused:     "⏸️ Posting paused: the bot can no longer post in this channel. Check its permissions, then pick the channel again.",
	BtnConfigChannel:  "📢  Set the channel",
	BtnConfigRole:     "🏷️  Set the role",
	BtnConfigMessage:  "🕒  Slots, display, language",
	ChannelModalTitle: "Set the channel",
	ChannelLabel:      "Availability channel",
	ChannelHint:       "Pick a text channel",
	RoleModalTitle:    "Set the role",
	RoleLabel:         "Role to mention",
	RoleHint:          "Pick the role to mention",
	MessageModalTitle: "Slots, display, language",
	LanguageLabel:     "Bot language",
	LanguageNames:     [2]string{"Français", "English"},
	DisplayLabel:      "Message display",
	DisplayHint:       "Changing the display or the language posts the weekly message again.",
	DisplayClassic:    "Classic (text)",
	DisplayImage:      "Image",
	ModeLabel:         "Day slots",
	ModeHint:          "⚠️ Changing the slots resets this week's availability.",
	ModeHourly:        "Hourly",
	StartHourLabel:    "First hour (hourly slots)",
	StartHourHint:     "Server time. Ignored for Afternoon / Evening.",
	EndHourLabel:      "End of the last slot (hourly slots)",
	EndHourHint:       "Ignored for Afternoon / Evening.",
	ConfigSaved:       "Settings saved.",
	NothingChanged:    "Nothing changed.",
	Republished:       "Settings saved: the weekly message was posted again.",
	SlotsReset:        "Slots updated: this week's availability was reset and the message posted again.",
	NotPublished:      " (no channel set: nothing was posted)",
	HourOrder:         "The first hour must be before the end of the last slot.",
	NothingSelected:   "Nothing was selected.",

	PanelPublication:      "📅  Posting",
	BtnConfigPublication:  "📅  Posting",
	PublicationModalTitle: "Weekly message posting",
	PublishDayLabel:       "Posting day",
	PublishDayHint:        "The message always covers a week from Monday to Sunday.",
	PublishSunday:         "Sunday, for the following week",
	PublishMonday:         "Monday, for the week starting",
	PublishHourLabel:      "Posting time",
	PublishHourHint:       "Server time.",
	PublicationSaved:      "Posting schedule saved: %s.",
	PublishedNow:          " The weekly message is being posted.",

	PanelTimezone:      "🌍  Timezone",
	BtnConfigTimezone:  "🌍  Timezone",
	GuildPickerContent: "🌍 **What's the server's timezone?** (currently: %s)\nAll its hours follow it: slots, grid, weeks and posting. Pick the current UTC offset, then the country.",
	GuildPickerCountry: "🌍 **Which country is the server in?** (%s)\nThe bot will follow daylight saving time changes by itself.",
	GuildTimezoneSaved: "Server timezone: **%s** (it is %s there).",
	TimezoneSlotsReset: " The slots moved, so this week's availability was reset.",

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
