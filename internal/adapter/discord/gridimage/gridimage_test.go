package gridimage

import (
	"sync"
	"testing"

	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
)

// Guilds render their grid in parallel (weekly job, refreshes): it must not share state.
func TestRenderConcurrently(t *testing.T) {
	view := appavailability.WeeklyView{
		Columns: []string{"20h", "21h", "22h", "23h", "0h"},
		Rows:    []string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"},
		Peak:    12,
	}
	for row := range view.Rows {
		view.Counts = append(view.Counts, []int{0, 3, 12, 7, 10})
		view.Closed = append(view.Closed, []bool{row == 0, row == 0, false, false, false})
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if img, err := Render(view); err != nil || len(img) == 0 {
				t.Errorf("Render: %d bytes, %v", len(img), err)
			}
		})
	}
	wg.Wait()
}
