package availability

// Level rates how full a slot is; the grid image shows it as a color.
type Level int

const (
	LevelEmpty Level = iota // nobody
	LevelLow                // a few players
	LevelMid                // getting there
	LevelHigh               // enough to play
)

// MaxFullAt bounds the players a guild can ask for a full slot (a select's options).
const MaxFullAt = 20

// MidAt returns the players from which a slot is LevelMid, for a guild whose slots are
// full from fullAt players: half of it, rounded up.
func MidAt(fullAt int) int {
	return (fullAt + 1) / 2
}

// SlotLevel rates a slot of count players. From fullAt players it is full; fullAt 0 rates
// it against the week's best slot instead (peak players): 75% of it is full, 40% mid.
func SlotLevel(count, peak, fullAt int) Level {
	switch {
	case count <= 0:
		return LevelEmpty
	case fullAt > 0 && count >= fullAt, fullAt <= 0 && count*4 >= peak*3:
		return LevelHigh
	case fullAt > 0 && count >= MidAt(fullAt), fullAt <= 0 && count*5 >= peak*2:
		return LevelMid
	default:
		return LevelLow
	}
}
