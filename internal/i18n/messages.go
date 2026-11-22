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
	DayPast         string // %s day
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
	AnnonceTeamDesc    string
	AnnonceRoleDesc    string

	// /help: the guild's teams, then setting one up
	TeamsTitle       string
	TeamsHint        string
	TeamsFooter      string // %d teams, %d maximum
	TeamPaused       string
	BtnCreateTeam    string
	BtnFirstTeam     string
	SetupTitle       string
	SetupIntro       string
	StepTeam         string
	StepChannel      string
	StepRole         string
	NextChannel      string
	RoleTip          string
	TeamReady        string // %s team
	TeamSetUp        string // %s team
	BtnBackToTeams   string
	BtnRename        string
	BtnPublishNow    string
	BtnDelete        string
	BtnConfirmDelete string
	BtnCancel        string
	DeleteTitle      string // %s team
	DeleteWarning    string
	CreateTitle      string
	RenameTitle      string
	TeamNameLabel    string
	TeamNameHint     string // %d characters
	TeamNameSample   string
	TeamCreated      string // %s team
	TeamRenamed      string // %s team
	TeamDeleted      string // %s team
	TeamGone         string
	NameEmpty        string
	NameTooLong      string // %d characters
	NameTaken        string
	TooManyTeams     string // %d teams
	PublishedIn      string // %s channel mention
	PublishFailed    string // %s channel mention

	// Message posted when the bot joins a guild
	WelcomeTitle   string // %s bot's name
	WelcomeIntro   string
	WelcomeStart   string
	WelcomeDone    string
	WelcomeFooter  string
	BtnStartSetup  string
	BtnManageTeams string
	BtnHowTo       string
	HowToTitle     string
	HowToBody      string

	// /help panel of a team: one section per setting, each with its button and form
	PanelFooter     string
	SectionPublish  string
	SectionSchedule string
	SectionSlots    string
	SectionTimezone string
	SectionMessage  string
	FieldChannel    string
	FieldRole       string
	FieldWhen       string
	FieldDisplay    string
	FieldLanguage   string
	NotConfigured   string
	NoRole          string
	PausedTitle     string
	PausedHint      string
	BtnEdit         string
	BtnSupport      string
	ConfigSaved     string
	NothingChanged  string
	PublishedNow    string

	// /help: channel and role form
	ChannelLabel        string
	ChannelHint         string
	NewChannelLabel     string
	NewChannelHint      string
	RoleLabel           string
	RoleHint            string
	NewRoleLabel        string
	NewRoleHint         string
	NeedManageChannels  string
	NeedManageRoles     string
	CreateChannelFailed string
	CreateRoleFailed    string
	CantPostThere       string
	RoleNotMentionable  string

	// /help: publication form
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

	// /help: timezone of the team
	GuildPickerContent string // %s current timezone
	GuildPickerCountry string // %s UTC offset
	GuildTimezoneSaved string // %s timezone name, %s current time there

	// /annonce
	AnnounceBy        string // %s mention
	AnnouncePublished string // %s channel mention
	AnnounceEmpty     string
	NoChannel         string // %s team
	NoTeam            string
	PickTeam          string
	UnknownTeam       string // %s team

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
	ServerTimeLegend: "Heure de l'équipe (%s) · %s = %s chez toi",
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
	DayPast:         "%s · passé",
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
	NoHoursToConvert:   "Les créneaux de cette équipe sont « Après-midi / Soir » : il n'y a pas d'heure à convertir.",
	HelpDescription:    "Panneau admin pour configurer les équipes",
	AnnonceDescription: "Publier une annonce (embed) dans le salon d'une équipe",
	AnnonceTitleDesc:   "Titre de l'annonce",
	AnnonceMessageDesc: "Contenu de l'annonce (utilise \\n pour un retour à la ligne)",
	AnnonceTeamDesc:    "Équipe concernée (facultatif s'il n'y en a qu'une)",
	AnnonceRoleDesc:    "Rôle à mentionner (par défaut : celui de l'équipe)",

	TeamsTitle:       "👥 Mes équipes",
	TeamsHint:        "Clique sur une équipe pour la configurer.",
	TeamsFooter:      "%d/%d équipes · 🔒 Réservé aux administrateurs",
	TeamPaused:       "en pause",
	BtnCreateTeam:    "➕ Créer une équipe",
	BtnFirstTeam:     "🚀 Créer ma première équipe",
	SetupTitle:       "🎯 Setup du serveur",
	SetupIntro:       "Chaque semaine, je publie un message de dispos pour chacune de tes équipes, et je mentionne leurs joueurs.",
	StepTeam:         "Créer ta première équipe",
	StepChannel:      "Choisir le salon des dispos",
	StepRole:         "Choisir un rôle à mentionner *(optionnel)*",
	NextChannel:      "👉 **Prochaine étape :** choisis le salon des dispos.",
	RoleTip:          "💡 *Optionnel :* choisis un rôle à mentionner pour prévenir tes joueurs.",
	TeamReady:        "🎉 **%s** est prête !",
	TeamSetUp:        " 🏆 Setup de **%s** terminé !",
	BtnBackToTeams:   "⬅️ Mes équipes",
	BtnRename:        "✏️ Renommer",
	BtnPublishNow:    "📤 Publier maintenant",
	BtnDelete:        "🗑️ Supprimer",
	BtnConfirmDelete: "🗑️ Oui, supprimer",
	BtnCancel:        "Annuler",
	DeleteTitle:      "🗑️ Supprimer %s ?",
	DeleteWarning:    "Ses dispos et son message de la semaine seront supprimés. Le salon et le rôle Discord sont conservés.",
	CreateTitle:      "➕ Créer une équipe",
	RenameTitle:      "✏️ Renommer l'équipe",
	TeamNameLabel:    "Nom de l'équipe",
	TeamNameHint:     "%d caractères maximum.",
	TeamNameSample:   "Ex : Team Phoenix",
	TeamCreated:      "✨ Équipe **%s** créée !",
	TeamRenamed:      "✏️ Équipe renommée en **%s**.",
	TeamDeleted:      "🗑️ Équipe **%s** supprimée.",
	TeamGone:         "Cette équipe n'existe plus.",
	NameEmpty:        "Le nom de l'équipe ne peut pas être vide.",
	NameTooLong:      "Le nom de l'équipe doit faire %d caractères maximum.",
	NameTaken:        "Une équipe porte déjà ce nom sur ce serveur.",
	TooManyTeams:     "Tu as atteint la limite de %d équipes.",
	PublishedIn:      "📤 Message publié dans %s.",
	PublishFailed:    "Impossible de publier dans %s : vérifie que j'ai accès au salon.",

	WelcomeTitle:   "👋 Heeey, moi c'est %s !",
	WelcomeIntro:   "Je suis là pour que ta team ne rate plus jamais une session 🎮\nChaque semaine, je poste les dispos, je ping tes joueurs, et en deux clics tout le monde sait **qui joue quand**. Plusieurs équipes ? Aucun souci, chacune a les siennes ! 🎨",
	WelcomeStart:   "👉 Un admin clique sur **🚀 Lancer le setup** (ou tape `/help`) et c'est parti !",
	WelcomeDone:    "🏆 **Setup terminé, bien joué !** Les dispos partent toutes seules chaque semaine.",
	WelcomeFooter:  "🔒 Le setup est réservé aux administrateurs",
	BtnStartSetup:  "🚀 Lancer le setup",
	BtnManageTeams: "⚙️ Gérer mes équipes",
	BtnHowTo:       "❓ Comment ça marche",
	HowToTitle:     "❓ Comment ça marche ?",
	HowToBody: "**1.** Un admin crée une ou plusieurs équipes avec `/help` : un nom, un salon et un rôle à mentionner.\n" +
		"**2.** Chaque semaine, je publie un message dans le salon de chaque équipe et je mentionne son rôle.\n" +
		"**3.** Les joueurs cliquent sur **✏️ Mes dispos** pour cocher leurs créneaux, et le message se met à jour en direct.\n" +
		"**4.** Chaque équipe a ses propres réglages : créneaux, affichage, langue, jour et heure de publication.\n\n" +
		"📣 `/annonce` publie aussi une annonce pour une équipe.",

	PanelFooter:     "🔒 Réservé aux administrateurs",
	SectionPublish:  "📢 Salon et rôle",
	SectionSchedule: "📅 Publication",
	SectionSlots:    "🕒 Créneaux",
	SectionTimezone: "🌍 Fuseau horaire",
	SectionMessage:  "🎨 Message",
	FieldChannel:    "Salon",
	FieldRole:       "Rôle",
	FieldWhen:       "Quand",
	FieldDisplay:    "Affichage",
	FieldLanguage:   "Langue",
	NotConfigured:   "❌ non configuré",
	NoRole:          "aucun",
	PausedTitle:     "⏸️ Publication en pause",
	PausedHint:      "Le bot ne peut plus publier dans ce salon. Vérifie ses permissions, puis choisis à nouveau le salon.",
	BtnEdit:         "✏️ Modifier",
	BtnSupport:      "💸 Soutenir le bot",
	ConfigSaved:     "Configuration enregistrée.",
	NothingChanged:  "Rien n'a changé.",
	PublishedNow:    " Le message de la semaine est en cours de publication.",

	ChannelLabel:        "Salon des dispos",
	ChannelHint:         "Où le message de la semaine est publié.",
	NewChannelLabel:     "…ou créer un nouveau salon",
	NewChannelHint:      "Tape un nom : je crée le salon, à la place du choix au-dessus.",
	RoleLabel:           "Rôle à mentionner",
	RoleHint:            "Optionnel : sans rôle, personne n'est mentionné.",
	NewRoleLabel:        "…ou créer un nouveau rôle",
	NewRoleHint:         "Tape un nom : je crée le rôle, à la place du choix au-dessus.",
	NeedManageChannels:  "Il me manque la permission **Gérer les salons** pour créer le salon. Ajoute-la à mon rôle ou choisis un salon existant.",
	NeedManageRoles:     "Il me manque la permission **Gérer les rôles** pour créer le rôle. Ajoute-la à mon rôle ou choisis un rôle existant.",
	CreateChannelFailed: "Impossible de créer le salon.",
	CreateRoleFailed:    "Impossible de créer le rôle.",
	CantPostThere:       "Je ne peux pas écrire dans ce salon : donne-moi l'accès (voir, envoyer des messages, intégrer des liens).",
	RoleNotMentionable:  "Ce rôle n'est pas mentionnable : sans la permission **Mentionner @everyone**, le ping ne notifiera personne.",

	PublishDayLabel:  "Jour de publication",
	PublishDayHint:   "Le message couvre toujours une semaine du lundi au dimanche.",
	PublishSunday:    "Dimanche, pour la semaine qui suit",
	PublishMonday:    "Lundi, pour la semaine qui commence",
	PublishHourLabel: "Heure de publication",
	PublishHourHint:  "Heure de l'équipe.",

	ModeLabel:        "Découpage des journées",
	ModeHint:         "⚠️ Changer le découpage réinitialise les dispos de la semaine en cours.",
	ModeHourly:       "Par heure",
	StartHourLabel:   "Première heure (mode par heure)",
	StartHourHint:    "Heure de l'équipe. Ignorée en Après-midi / Soir.",
	EndHourLabel:     "Fin du dernier créneau (mode par heure)",
	EndHourHint:      "Jusqu'à 5h le lendemain. Ignorée en Après-midi / Soir.",
	NextDay:          " (lendemain)",
	HourOrder:        "La première heure doit être avant la fin du dernier créneau.",
	MemberTimeLabel:  "Heures des membres (mode par heure)",
	MemberTimeHint:   "International : chacun indique son fuseau et voit les créneaux à son heure.",
	MemberTimeServer: "🏠 Tout le monde à l'heure de l'équipe",
	MemberTimeOwn:    "🌍 International : chacun à son heure",
	FieldMembers:     "Membres",
	MembersServer:    "à l'heure de l'équipe",
	MembersOwn:       "chacun à son heure",
	MemberTimeOff:    "Dans cette équipe, tout le monde voit les créneaux à l'heure de l'équipe.",
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

	GuildPickerContent: "🌍 **Quel est le fuseau horaire de l'équipe ?** (actuellement : %s)\nToutes ses heures le suivent : créneaux, grille, semaines et publication. Choisis le décalage UTC actuel, puis le pays.",
	GuildPickerCountry: "🌍 **Dans quel pays est l'équipe ?** (%s)\nLe bot suivra tout seul les passages à l'heure d'été et d'hiver.",
	GuildTimezoneSaved: "Fuseau de l'équipe : **%s** (il y est %s).",

	AnnounceBy:        "**Annonce de** %s",
	AnnouncePublished: "✅ Annonce publiée dans %s.",
	AnnounceEmpty:     "Le titre et le message ne peuvent pas être vides.",
	NoChannel:         "L'équipe **%s** n'a pas de salon. Configure-le avec `/help`.",
	NoTeam:            "Aucune équipe sur ce serveur. Crée-en une avec `/help`.",
	PickTeam:          "Ce serveur a plusieurs équipes : précise laquelle avec l'option `equipe`.",
	UnknownTeam:       "Aucune équipe ne s'appelle **%s**.",

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
	ServerTimeLegend: "Team time (%s) · %s = %s for you",
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
	DayPast:         "%s · over",
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
	NoHoursToConvert:   "This team uses \"Afternoon / Evening\" slots: there are no hours to convert.",
	HelpDescription:    "Admin panel to configure the teams",
	AnnonceDescription: "Post an announcement (embed) in a team's channel",
	AnnonceTitleDesc:   "Title of the announcement",
	AnnonceMessageDesc: "Content of the announcement (use \\n for a line break)",
	AnnonceTeamDesc:    "Team concerned (optional when there is only one)",
	AnnonceRoleDesc:    "Role to mention (by default: the team's)",

	TeamsTitle:       "👥 My teams",
	TeamsHint:        "Click a team to set it up.",
	TeamsFooter:      "%d/%d teams · 🔒 Admins only",
	TeamPaused:       "paused",
	BtnCreateTeam:    "➕ Create a team",
	BtnFirstTeam:     "🚀 Create my first team",
	SetupTitle:       "🎯 Server setup",
	SetupIntro:       "Every week, I post an availability message for each of your teams and mention their players.",
	StepTeam:         "Create your first team",
	StepChannel:      "Pick the availability channel",
	StepRole:         "Pick a role to mention *(optional)*",
	NextChannel:      "👉 **Next step:** pick the availability channel.",
	RoleTip:          "💡 *Optional:* pick a role to mention to notify your players.",
	TeamReady:        "🎉 **%s** is ready!",
	TeamSetUp:        " 🏆 **%s** is all set up!",
	BtnBackToTeams:   "⬅️ My teams",
	BtnRename:        "✏️ Rename",
	BtnPublishNow:    "📤 Post now",
	BtnDelete:        "🗑️ Delete",
	BtnConfirmDelete: "🗑️ Yes, delete",
	BtnCancel:        "Cancel",
	DeleteTitle:      "🗑️ Delete %s?",
	DeleteWarning:    "Its availability and weekly message will be deleted. The Discord channel and role are kept.",
	CreateTitle:      "➕ Create a team",
	RenameTitle:      "✏️ Rename the team",
	TeamNameLabel:    "Team name",
	TeamNameHint:     "%d characters at most.",
	TeamNameSample:   "E.g. Team Phoenix",
	TeamCreated:      "✨ Team **%s** created!",
	TeamRenamed:      "✏️ Team renamed to **%s**.",
	TeamDeleted:      "🗑️ Team **%s** deleted.",
	TeamGone:         "This team no longer exists.",
	NameEmpty:        "The team's name can't be empty.",
	NameTooLong:      "The team's name must be %d characters at most.",
	NameTaken:        "A team already has this name on this server.",
	TooManyTeams:     "You reached the limit of %d teams.",
	PublishedIn:      "📤 Message posted in %s.",
	PublishFailed:    "Couldn't post in %s: check that I can access the channel.",

	WelcomeTitle:   "👋 Heeey, I'm %s!",
	WelcomeIntro:   "I'm here so your team never misses a session again 🎮\nEvery week, I post the availability, ping your players, and in two clicks everyone knows **who plays when**. Several teams? No problem, each gets its own! 🎨",
	WelcomeStart:   "👉 An admin clicks **🚀 Start the setup** (or types `/help`) and off we go!",
	WelcomeDone:    "🏆 **Setup done, nice one!** The availability goes out by itself every week.",
	WelcomeFooter:  "🔒 The setup is for admins only",
	BtnStartSetup:  "🚀 Start the setup",
	BtnManageTeams: "⚙️ Manage my teams",
	BtnHowTo:       "❓ How it works",
	HowToTitle:     "❓ How it works",
	HowToBody: "**1.** An admin creates one or more teams with `/help`: a name, a channel and a role to mention.\n" +
		"**2.** Every week, I post a message in each team's channel and mention its role.\n" +
		"**3.** Players click **✏️ My availability** to pick their slots, and the message updates live.\n" +
		"**4.** Each team has its own settings: slots, display, language, posting day and time.\n\n" +
		"📣 `/annonce` also posts an announcement for a team.",

	PanelFooter:     "🔒 Admins only",
	SectionPublish:  "📢 Channel and role",
	SectionSchedule: "📅 Posting",
	SectionSlots:    "🕒 Slots",
	SectionTimezone: "🌍 Timezone",
	SectionMessage:  "🎨 Message",
	FieldChannel:    "Channel",
	FieldRole:       "Role",
	FieldWhen:       "When",
	FieldDisplay:    "Display",
	FieldLanguage:   "Language",
	NotConfigured:   "❌ not set",
	NoRole:          "none",
	PausedTitle:     "⏸️ Posting paused",
	PausedHint:      "The bot can no longer post in this channel. Check its permissions, then pick the channel again.",
	BtnEdit:         "✏️ Edit",
	BtnSupport:      "💸 Support the bot",
	ConfigSaved:     "Settings saved.",
	NothingChanged:  "Nothing changed.",
	PublishedNow:    " The weekly message is being posted.",

	ChannelLabel:        "Availability channel",
	ChannelHint:         "Where the weekly message is posted.",
	NewChannelLabel:     "…or create a new channel",
	NewChannelHint:      "Type a name: I create the channel, instead of the choice above.",
	RoleLabel:           "Role to mention",
	RoleHint:            "Optional: without a role, nobody is mentioned.",
	NewRoleLabel:        "…or create a new role",
	NewRoleHint:         "Type a name: I create the role, instead of the choice above.",
	NeedManageChannels:  "I'm missing the **Manage Channels** permission to create the channel. Add it to my role or pick an existing channel.",
	NeedManageRoles:     "I'm missing the **Manage Roles** permission to create the role. Add it to my role or pick an existing role.",
	CreateChannelFailed: "Couldn't create the channel.",
	CreateRoleFailed:    "Couldn't create the role.",
	CantPostThere:       "I can't write in this channel: give me access (view, send messages, embed links).",
	RoleNotMentionable:  "This role isn't mentionable: without the **Mention @everyone** permission, the ping won't notify anyone.",

	PublishDayLabel:  "Posting day",
	PublishDayHint:   "The message always covers a week from Monday to Sunday.",
	PublishSunday:    "Sunday, for the following week",
	PublishMonday:    "Monday, for the week starting",
	PublishHourLabel: "Posting time",
	PublishHourHint:  "Team time.",

	ModeLabel:        "Day slots",
	ModeHint:         "⚠️ Changing the slots resets this week's availability.",
	ModeHourly:       "Hourly",
	StartHourLabel:   "First hour (hourly slots)",
	StartHourHint:    "Team time. Ignored for Afternoon / Evening.",
	EndHourLabel:     "End of the last slot (hourly slots)",
	EndHourHint:      "Up to 5 AM the next day. Ignored for Afternoon / Evening.",
	NextDay:          " (next day)",
	HourOrder:        "The first hour must be before the end of the last slot.",
	MemberTimeLabel:  "Members' hours (hourly slots)",
	MemberTimeHint:   "International: everyone sets their timezone and sees the slots in their own time.",
	MemberTimeServer: "🏠 Everyone on the team's time",
	MemberTimeOwn:    "🌍 International: everyone in their own time",
	FieldMembers:     "Members",
	MembersServer:    "on the team's time",
	MembersOwn:       "each in their own time",
	MemberTimeOff:    "In this team, everyone sees the slots in the team's time.",
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

	GuildPickerContent: "🌍 **What's the team's timezone?** (currently: %s)\nAll its hours follow it: slots, grid, weeks and posting. Pick the current UTC offset, then the country.",
	GuildPickerCountry: "🌍 **Which country is the team in?** (%s)\nThe bot will follow daylight saving time changes by itself.",
	GuildTimezoneSaved: "Team timezone: **%s** (it is %s there).",

	AnnounceBy:        "**Announcement by** %s",
	AnnouncePublished: "✅ Announcement posted in %s.",
	AnnounceEmpty:     "The title and the message can't be empty.",
	NoChannel:         "Team **%s** has no channel. Set it with `/help`.",
	NoTeam:            "No team on this server. Create one with `/help`.",
	PickTeam:          "This server has several teams: say which one with the `equipe` option.",
	UnknownTeam:       "No team is called **%s**.",

	ErrGeneric:     "Something went wrong, please try again in a moment.",
	ErrUnknownUser: "Couldn't determine your Discord user for this interaction.",
	ErrInvalid:     "Invalid interaction.",
	ErrAdminOnly:   "You must be a server admin or the bot's creator to use this.",
	ErrGuildOnly:   "This command must be used in a server.",
}
