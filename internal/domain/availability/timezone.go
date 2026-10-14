package availability

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // countries' timezones and DefaultTimezone load even on hosts without tzdata
)

// DefaultTimezone is the timezone of the guilds that never chose one.
const DefaultTimezone = "Europe/Paris"

// DefaultLocation returns DefaultTimezone, loaded once (tzdata is embedded: it can't fail).
var DefaultLocation = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		panic(fmt.Sprintf("load %s from embedded tzdata: %v", DefaultTimezone, err))
	}
	return loc
})

// Timezone ties a place, named in French and English, to its IANA timezone.
type Timezone struct {
	Zone string // IANA name, e.g. "Europe/Paris"
	Fr   string // e.g. "Allemagne"
	En   string // e.g. "Germany"
}

// Countries maps places to their timezone, francophone places first: they come first in
// the picker descriptions. Offsets are never stored here, they are computed from tzdata at
// the current date, so daylight saving time moves a country to the right offset by itself.
var Countries = []Timezone{
	// Francophonie
	{"Europe/Paris", "France", "France"},
	{"Europe/Brussels", "Belgique", "Belgium"},
	{"Europe/Zurich", "Suisse", "Switzerland"},
	{"Europe/Luxembourg", "Luxembourg", "Luxembourg"},
	{"America/Toronto", "Québec", "Quebec"},
	{"America/Martinique", "Martinique", "Martinique"},
	{"America/Guadeloupe", "Guadeloupe", "Guadeloupe"},
	{"America/Cayenne", "Guyane", "French Guiana"},
	{"America/Miquelon", "Saint-Pierre-et-Miquelon", "Saint Pierre and Miquelon"},
	{"America/Port-au-Prince", "Haïti", "Haiti"},
	{"Indian/Reunion", "La Réunion", "Réunion"},
	{"Indian/Mayotte", "Mayotte", "Mayotte"},
	{"Indian/Mauritius", "Maurice", "Mauritius"},
	{"Indian/Antananarivo", "Madagascar", "Madagascar"},
	{"Pacific/Noumea", "Nouvelle-Calédonie", "New Caledonia"},
	{"Pacific/Tahiti", "Tahiti", "Tahiti"},
	{"Pacific/Marquesas", "Marquises", "Marquesas Islands"},
	{"Pacific/Wallis", "Wallis-et-Futuna", "Wallis and Futuna"},
	{"Africa/Casablanca", "Maroc", "Morocco"},
	{"Africa/Algiers", "Algérie", "Algeria"},
	{"Africa/Tunis", "Tunisie", "Tunisia"},
	{"Africa/Dakar", "Sénégal", "Senegal"},
	{"Africa/Abidjan", "Côte d'Ivoire", "Ivory Coast"},
	{"Africa/Douala", "Cameroun", "Cameroon"},
	{"Africa/Kinshasa", "RD Congo", "DR Congo"},
	// Europe
	{"Europe/London", "Royaume-Uni", "United Kingdom"},
	{"Europe/Dublin", "Irlande", "Ireland"},
	{"Europe/Lisbon", "Portugal", "Portugal"},
	{"Atlantic/Reykjavik", "Islande", "Iceland"},
	{"Europe/Madrid", "Espagne", "Spain"},
	{"Europe/Rome", "Italie", "Italy"},
	{"Europe/Berlin", "Allemagne", "Germany"},
	{"Europe/Amsterdam", "Pays-Bas", "Netherlands"},
	{"Europe/Warsaw", "Pologne", "Poland"},
	{"Europe/Stockholm", "Suède", "Sweden"},
	{"Europe/Athens", "Grèce", "Greece"},
	{"Europe/Helsinki", "Finlande", "Finland"},
	{"Europe/Bucharest", "Roumanie", "Romania"},
	{"Europe/Kyiv", "Ukraine", "Ukraine"},
	{"Europe/Istanbul", "Turquie", "Turkey"},
	{"Europe/Moscow", "Russie (Moscou)", "Russia (Moscow)"},
	{"Atlantic/Azores", "Açores", "Azores"},
	{"Atlantic/Cape_Verde", "Cap-Vert", "Cape Verde"},
	// Afrique, Moyen-Orient
	{"Africa/Cairo", "Égypte", "Egypt"},
	{"Africa/Johannesburg", "Afrique du Sud", "South Africa"},
	{"Asia/Jerusalem", "Israël", "Israel"},
	{"Asia/Riyadh", "Arabie saoudite", "Saudi Arabia"},
	{"Asia/Tehran", "Iran", "Iran"},
	{"Asia/Dubai", "Émirats arabes unis", "United Arab Emirates"},
	{"Asia/Kabul", "Afghanistan", "Afghanistan"},
	// Asie
	{"Asia/Karachi", "Pakistan", "Pakistan"},
	{"Asia/Kolkata", "Inde", "India"},
	{"Asia/Kathmandu", "Népal", "Nepal"},
	{"Asia/Dhaka", "Bangladesh", "Bangladesh"},
	{"Asia/Yangon", "Myanmar", "Myanmar"},
	{"Asia/Bangkok", "Thaïlande", "Thailand"},
	{"Asia/Ho_Chi_Minh", "Vietnam", "Vietnam"},
	{"Asia/Jakarta", "Indonésie (Jakarta)", "Indonesia (Jakarta)"},
	{"Asia/Shanghai", "Chine", "China"},
	{"Asia/Singapore", "Singapour", "Singapore"},
	{"Asia/Manila", "Philippines", "Philippines"},
	{"Asia/Taipei", "Taïwan", "Taiwan"},
	{"Asia/Hong_Kong", "Hong Kong", "Hong Kong"},
	{"Asia/Tokyo", "Japon", "Japan"},
	{"Asia/Seoul", "Corée du Sud", "South Korea"},
	{"Asia/Kamchatka", "Russie (Kamtchatka)", "Russia (Kamchatka)"},
	// Océanie
	{"Australia/Perth", "Australie (Perth)", "Australia (Perth)"},
	{"Australia/Darwin", "Australie (Darwin)", "Australia (Darwin)"},
	{"Australia/Adelaide", "Australie (Adélaïde)", "Australia (Adelaide)"},
	{"Australia/Brisbane", "Australie (Brisbane)", "Australia (Brisbane)"},
	{"Australia/Sydney", "Australie (Sydney)", "Australia (Sydney)"},
	{"Australia/Lord_Howe", "Île Lord Howe", "Lord Howe Island"},
	{"Pacific/Guadalcanal", "Îles Salomon", "Solomon Islands"},
	{"Pacific/Auckland", "Nouvelle-Zélande", "New Zealand"},
	{"Pacific/Fiji", "Fidji", "Fiji"},
	{"Pacific/Tongatapu", "Tonga", "Tonga"},
	{"Pacific/Apia", "Samoa", "Samoa"},
	{"Pacific/Kiritimati", "Kiribati (Line)", "Kiribati (Line Islands)"},
	{"Pacific/Pago_Pago", "Samoa américaines", "American Samoa"},
	{"Pacific/Niue", "Niue", "Niue"},
	{"Pacific/Honolulu", "Hawaï", "Hawaii"},
	{"Etc/GMT+12", "Îles Baker et Howland (inhabitées)", "Baker and Howland Islands (uninhabited)"}, // Etc/GMT+12 is UTC-12 (POSIX sign)
	// Amériques
	{"America/Anchorage", "Alaska", "Alaska"},
	{"America/Los_Angeles", "USA (Los Angeles)", "USA (Los Angeles)"},
	{"America/Vancouver", "Canada (Vancouver)", "Canada (Vancouver)"},
	{"America/Phoenix", "USA (Arizona)", "USA (Arizona)"},
	{"America/Denver", "USA (Denver)", "USA (Denver)"},
	{"America/Mexico_City", "Mexique", "Mexico"},
	{"America/Chicago", "USA (Chicago, Texas)", "USA (Chicago, Texas)"},
	{"America/New_York", "USA (New York)", "USA (New York)"},
	{"America/Bogota", "Colombie", "Colombia"},
	{"America/Lima", "Pérou", "Peru"},
	{"America/Caracas", "Venezuela", "Venezuela"},
	{"America/Santiago", "Chili", "Chile"},
	{"America/St_Johns", "Terre-Neuve", "Newfoundland"},
	{"America/Sao_Paulo", "Brésil (São Paulo)", "Brazil (São Paulo)"},
	{"America/Argentina/Buenos_Aires", "Argentine", "Argentina"},
	{"America/Montevideo", "Uruguay", "Uruguay"},
	{"America/Noronha", "Brésil (Noronha)", "Brazil (Noronha)"},
	{"America/Nuuk", "Groenland", "Greenland"},
}

// countryLocations are the locations of Countries, loaded once (LoadLocation reads and
// parses tzdata on every call). A zone missing from tzdata is nil.
var countryLocations = sync.OnceValue(func() []*time.Location {
	locs := make([]*time.Location, len(Countries))
	for i, c := range Countries {
		locs[i], _ = time.LoadLocation(c.Zone)
	}
	return locs
})

// CountriesByOffset groups the places of Countries by their UTC offset (in seconds) at the
// given time, in list order.
func CountriesByOffset(at time.Time) map[int][]Timezone {
	byOffset := make(map[int][]Timezone)
	for i, loc := range countryLocations() {
		if loc == nil {
			continue
		}
		_, offset := at.In(loc).Zone()
		byOffset[offset] = append(byOffset[offset], Countries[i])
	}
	return byOffset
}

var utcOffsetPattern = regexp.MustCompile(`^(?:UTC|GMT)?\s*([+-])\s*(\d{1,2})(?::?(\d{2}))?$`)

// ParseTimezone accepts a fixed offset ("UTC+5:30", "GMT-3", "+2"), as offered by the
// timezone picker, or an IANA name ("America/Sao_Paulo").
func ParseTimezone(input string) (*time.Location, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty timezone")
	}
	if m := utcOffsetPattern.FindStringSubmatch(strings.ToUpper(input)); m != nil {
		hours, _ := strconv.Atoi(m[2])
		minutes := 0
		if m[3] != "" {
			minutes, _ = strconv.Atoi(m[3])
		}
		if hours > 14 || minutes >= 60 {
			return nil, fmt.Errorf("invalid UTC offset %q", input)
		}
		offset := hours*3600 + minutes*60
		if m[1] == "-" {
			offset = -offset
		}
		return time.FixedZone(FormatUTCOffset(offset), offset), nil
	}
	if strings.EqualFold(input, "UTC") || strings.EqualFold(input, "GMT") {
		return time.UTC, nil
	}
	return time.LoadLocation(input)
}

// FormatUTCOffset formats an offset in seconds as "UTC+5:30" / "UTC-3".
func FormatUTCOffset(offset int) string {
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	if m := offset % 3600 / 60; m != 0 {
		return fmt.Sprintf("UTC%s%d:%02d", sign, offset/3600, m)
	}
	return fmt.Sprintf("UTC%s%d", sign, offset/3600)
}
