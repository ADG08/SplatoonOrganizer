package availability

import "testing"

func TestSlotLevel(t *testing.T) {
	for _, c := range []struct {
		count, peak, fullAt int
		want                Level
	}{
		{0, 5, 0, LevelEmpty},
		// Relative to the week's best slot (the default).
		{5, 5, 0, LevelHigh},
		{4, 5, 0, LevelHigh}, // 80% of the best
		{2, 5, 0, LevelMid},  // 40%
		{1, 5, 0, LevelLow},
		{2, 2, 0, LevelHigh}, // a quiet week: 2 players is the best, so green
		// Full from 4 players, as a guild may ask.
		{2, 2, 4, LevelMid}, // the same quiet week isn't green anymore
		{4, 9, 4, LevelHigh},
		{9, 9, 4, LevelHigh},
		{3, 9, 4, LevelMid},
		{1, 9, 4, LevelLow},
		{1, 1, 1, LevelHigh},
	} {
		if got := SlotLevel(c.count, c.peak, c.fullAt); got != c.want {
			t.Errorf("SlotLevel(%d, peak %d, full at %d) = %d, want %d", c.count, c.peak, c.fullAt, got, c.want)
		}
	}
	for fullAt, want := range map[int]int{1: 1, 2: 1, 3: 2, 4: 2, 5: 3} {
		if got := MidAt(fullAt); got != want {
			t.Errorf("MidAt(%d) = %d, want %d", fullAt, got, want)
		}
	}
}
