package weekly

// Custom IDs of the weekly message's components, handled by package handlers. They are
// stored in the messages already posted: changing one breaks the buttons of those.
const (
	OpenDisposCustomID  = "availability:open"
	ViewDisposCustomID  = "availability:view"
	WhoDisposCustomID   = "availability:who"
	MyTimeCustomID      = "availability:mytime"
	UnavailableCustomID = "availability:unavailable"

	// WhoAllValue is the "who is available" option showing the whole week.
	WhoAllValue = "all"
)
