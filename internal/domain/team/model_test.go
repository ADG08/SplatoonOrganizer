package team

import (
	"errors"
	"strings"
	"testing"
)

func TestStepsDone(t *testing.T) {
	for _, c := range []struct {
		team     Team
		want     int
		complete bool
	}{
		{Team{}, 1, false},
		{Team{ChannelID: "c"}, 2, true},
		{Team{RoleID: "r"}, 1, false}, // the role is optional: it doesn't count
		{Team{ChannelID: "c", RoleID: "r"}, 2, true},
	} {
		if got := c.team.StepsDone(); got != c.want {
			t.Errorf("%+v: %d steps done, want %d", c.team, got, c.want)
		}
		if got := c.team.IsComplete(); got != c.complete {
			t.Errorf("%+v: complete %v, want %v", c.team, got, c.complete)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	for _, c := range []struct {
		in, want string
		err      error
	}{
		{"  Team A  ", "Team A", nil},
		{strings.Repeat("é", MaxNameLength), strings.Repeat("é", MaxNameLength), nil},
		{"", "", ErrNameEmpty},
		{"   ", "", ErrNameEmpty},
		{strings.Repeat("a", MaxNameLength+1), "", ErrNameTooLong},
	} {
		got, err := NormalizeName(c.in)
		if !errors.Is(err, c.err) || got != c.want {
			t.Errorf("NormalizeName(%q) = %q, %v, want %q, %v", c.in, got, err, c.want, c.err)
		}
	}
}
