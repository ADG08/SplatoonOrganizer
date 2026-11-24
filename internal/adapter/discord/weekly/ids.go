package weekly

import (
	"fmt"
	"strconv"
	"strings"
)

// Custom IDs of the weekly message's components, handled by package handlers, followed by
// the team's ID (see CustomID). They are stored in the messages already posted: changing
// one breaks the buttons of those.
const (
	OpenDisposCustomID  = "availability:open"
	ViewDisposCustomID  = "availability:view"
	WhoDisposCustomID   = "availability:who"
	MyTimeCustomID      = "availability:mytime"
	UnavailableCustomID = "availability:unavailable"

	// WhoAllValue is the "who is available" option showing the whole week.
	WhoAllValue = "all"
)

// CustomID builds the custom ID of a team's component: <prefix>:<teamID>[:<rest>...].
func CustomID(prefix string, teamID int64, rest ...string) string {
	return strings.Join(append([]string{prefix, strconv.FormatInt(teamID, 10)}, rest...), ":")
}

// ParseCustomID reads a custom ID built by CustomID, and returns what follows the team's
// ID. The bare prefix of a message posted before teams existed gives team ID 0.
func ParseCustomID(customID, prefix string) (teamID int64, rest string, err error) {
	if customID == prefix {
		return 0, "", nil
	}
	id, ok := strings.CutPrefix(customID, prefix+":")
	if !ok {
		return 0, "", fmt.Errorf("custom id %q: no prefix %q", customID, prefix)
	}
	id, rest, _ = strings.Cut(id, ":")
	if teamID, err = strconv.ParseInt(id, 10, 64); err != nil || teamID <= 0 {
		return 0, "", fmt.Errorf("custom id %q: invalid team id", customID)
	}
	return teamID, rest, nil
}
