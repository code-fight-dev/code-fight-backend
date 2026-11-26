package tournament

import (
	"slices"
	"testing"
)

func TestContestRatingChanges(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field []ratedParticipant
		want  []int
	}{
		{"win", []ratedParticipant{{Rank: 1, Rating: 1000}, {Rank: 2, Rating: 1000}}, []int{16, -16}},
		{"upset", []ratedParticipant{{Rank: 1, Rating: 1000}, {Rank: 2, Rating: 1400}}, []int{29, -29}},
		{"shared first", []ratedParticipant{{Rank: 1, Rating: 1000}, {Rank: 1, Rating: 1000}, {Rank: 2, Rating: 1000}}, []int{8, 8, -16}},
		{"single player", []ratedParticipant{{Rank: 1, Rating: 1000}}, []int{0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := contestRatingChanges(tc.field); !slices.Equal(got, tc.want) {
				t.Fatalf("rating changes = %v, want %v", got, tc.want)
			}
		})
	}
}
