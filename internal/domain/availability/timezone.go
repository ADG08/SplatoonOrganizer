package availability

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Timezone ties a place to its IANA timezone.
type Timezone struct {
	Zone  string // IANA name, e.g. "Europe/Paris"
	Where string // e.g. "France"
}

// Countries maps places to their timezone, francophone places first: they come first in
// the picker descriptions. Offsets are never stored here, they are computed from tzdata at
// the current date, so daylight saving time moves a country to the right offset by itself.
var Countries = []Timezone{
	// Francophonie
	{"Europe/Paris", "France"},
	{"Europe/Brussels", "Belgique"},
	{"Europe/Zurich", "Suisse"},
	{"Europe/Luxembourg", "Luxembourg"},
	{"America/Toronto", "Québec"},
	{"America/Martinique", "Martinique"},
	{"America/Guadeloupe", "Guadeloupe"},
	{"America/Cayenne", "Guyane"},
	{"America/Miquelon", "Saint-Pierre-et-Miquelon"},
	{"America/Port-au-Prince", "Haïti"},
	{"Indian/Reunion", "La Réunion"},
	{"Indian/Mayotte", "Mayotte"},
	{"Indian/Mauritius", "Maurice"},
	{"Indian/Antananarivo", "Madagascar"},
	{"Pacific/Noumea", "Nouvelle-Calédonie"},
	{"Pacific/Tahiti", "Tahiti"},
	{"Pacific/Marquesas", "Marquises"},
	{"Pacific/Wallis", "Wallis-et-Futuna"},
	{"Africa/Casablanca", "Maroc"},
	{"Africa/Algiers", "Algérie"},
	{"Africa/Tunis", "Tunisie"},
	{"Africa/Dakar", "Sénégal"},
	{"Africa/Abidjan", "Côte d'Ivoire"},
	{"Africa/Douala", "Cameroun"},
	{"Africa/Kinshasa", "RD Congo"},
	// Europe
	{"Europe/London", "Royaume-Uni"},
	{"Europe/Dublin", "Irlande"},
	{"Europe/Lisbon", "Portugal"},
	{"Atlantic/Reykjavik", "Islande"},
	{"Europe/Madrid", "Espagne"},
	{"Europe/Rome", "Italie"},
	{"Europe/Berlin", "Allemagne"},
	{"Europe/Amsterdam", "Pays-Bas"},
	{"Europe/Warsaw", "Pologne"},
	{"Europe/Stockholm", "Suède"},
	{"Europe/Athens", "Grèce"},
	{"Europe/Helsinki", "Finlande"},
	{"Europe/Bucharest", "Roumanie"},
	{"Europe/Kyiv", "Ukraine"},
	{"Europe/Istanbul", "Turquie"},
	{"Europe/Moscow", "Russie (Moscou)"},
	{"Atlantic/Azores", "Açores"},
	{"Atlantic/Cape_Verde", "Cap-Vert"},
	// Afrique, Moyen-Orient
	{"Africa/Cairo", "Égypte"},
	{"Africa/Johannesburg", "Afrique du Sud"},
	{"Asia/Jerusalem", "Israël"},
	{"Asia/Riyadh", "Arabie saoudite"},
	{"Asia/Tehran", "Iran"},
	{"Asia/Dubai", "Émirats arabes unis"},
	{"Asia/Kabul", "Afghanistan"},
	// Asie
	{"Asia/Karachi", "Pakistan"},
	{"Asia/Kolkata", "Inde"},
	{"Asia/Kathmandu", "Népal"},
	{"Asia/Dhaka", "Bangladesh"},
	{"Asia/Yangon", "Myanmar"},
	{"Asia/Bangkok", "Thaïlande"},
	{"Asia/Ho_Chi_Minh", "Vietnam"},
	{"Asia/Jakarta", "Indonésie (Jakarta)"},
	{"Asia/Shanghai", "Chine"},
	{"Asia/Singapore", "Singapour"},
	{"Asia/Manila", "Philippines"},
	{"Asia/Taipei", "Taïwan"},
	{"Asia/Hong_Kong", "Hong Kong"},
	{"Asia/Tokyo", "Japon"},
	{"Asia/Seoul", "Corée du Sud"},
	{"Asia/Kamchatka", "Russie (Kamtchatka)"},
	// Océanie
	{"Australia/Perth", "Australie (Perth)"},
	{"Australia/Darwin", "Australie (Darwin)"},
	{"Australia/Adelaide", "Australie (Adélaïde)"},
	{"Australia/Brisbane", "Australie (Brisbane)"},
	{"Australia/Sydney", "Australie (Sydney)"},
	{"Australia/Lord_Howe", "Île Lord Howe"},
	{"Pacific/Guadalcanal", "Îles Salomon"},
	{"Pacific/Auckland", "Nouvelle-Zélande"},
	{"Pacific/Fiji", "Fidji"},
	{"Pacific/Tongatapu", "Tonga"},
	{"Pacific/Apia", "Samoa"},
	{"Pacific/Kiritimati", "Kiribati (Line)"},
	{"Pacific/Pago_Pago", "Samoa américaines"},
	{"Pacific/Niue", "Niue"},
	{"Pacific/Honolulu", "Hawaï"},
	{"Etc/GMT+12", "Îles Baker et Howland (inhabitées)"}, // Etc/GMT+12 is UTC-12 (POSIX sign)
	// Amériques
	{"America/Anchorage", "Alaska"},
	{"America/Los_Angeles", "USA (Los Angeles)"},
	{"America/Vancouver", "Canada (Vancouver)"},
	{"America/Phoenix", "USA (Arizona)"},
	{"America/Denver", "USA (Denver)"},
	{"America/Mexico_City", "Mexique"},
	{"America/Chicago", "USA (Chicago, Texas)"},
	{"America/New_York", "USA (New York)"},
	{"America/Bogota", "Colombie"},
	{"America/Lima", "Pérou"},
	{"America/Caracas", "Venezuela"},
	{"America/Santiago", "Chili"},
	{"America/St_Johns", "Terre-Neuve"},
	{"America/Sao_Paulo", "Brésil (São Paulo)"},
	{"America/Argentina/Buenos_Aires", "Argentine"},
	{"America/Montevideo", "Uruguay"},
	{"America/Noronha", "Brésil (Noronha)"},
	{"America/Nuuk", "Groenland"},
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
func CountriesByOffset(at time.Time) map[int][]string {
	byOffset := make(map[int][]string)
	for i, loc := range countryLocations() {
		if loc == nil {
			continue
		}
		_, offset := at.In(loc).Zone()
		byOffset[offset] = append(byOffset[offset], Countries[i].Where)
	}
	return byOffset
}

var utcOffsetPattern = regexp.MustCompile(`^(?:UTC|GMT)?\s*([+-])\s*(\d{1,2})(?::?(\d{2}))?$`)

// ParseTimezone accepts an IANA name ("America/Sao_Paulo", case-insensitive for the
// zones of Countries) or a fixed offset ("UTC+5:30", "GMT-3", "+2").
func ParseTimezone(input string) (*time.Location, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty timezone")
	}
	for _, tz := range Countries {
		if strings.EqualFold(tz.Zone, input) {
			return time.LoadLocation(tz.Zone)
		}
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

// TimezoneName describes a location for members: the place of Countries when known,
// else the IANA name or the UTC offset.
func TimezoneName(loc *time.Location) string {
	for _, tz := range Countries {
		if tz.Zone == loc.String() {
			return tz.Where
		}
	}
	return loc.String()
}

// ClockLabel formats a time of day the French way: "20h", "20h30".
func ClockLabel(t time.Time) string {
	if t.Minute() == 0 {
		return fmt.Sprintf("%dh", t.Hour())
	}
	return fmt.Sprintf("%dh%02d", t.Hour(), t.Minute())
}
