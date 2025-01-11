package token_test

import (
	"strconv"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot/token"
)

func TestPosition(t *testing.T) {
	pos := token.Position{Row: 2, Column: 2}
	tests := []struct {
		in   token.Position
		want map[string]bool
	}{
		{
			in: token.Position{Row: 1, Column: 1},
			want: map[string]bool{
				"Before": false,
				"After":  true,
			},
		},
		{
			in: token.Position{Row: 2, Column: 1},
			want: map[string]bool{
				"Before": false,
				"After":  true,
			},
		},
		{
			in: token.Position{Row: 2, Column: 2},
			want: map[string]bool{
				"Before": false,
				"After":  false,
			},
		},
		{
			in: token.Position{Row: 2, Column: 3},
			want: map[string]bool{
				"Before": true,
				"After":  false,
			},
		},
		{
			in: token.Position{Row: 3, Column: 1},
			want: map[string]bool{
				"Before": true,
				"After":  false,
			},
		},
	}
	t.Run("Before", func(t *testing.T) {
		for i, test := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				got := pos.Before(test.in)

				assert.Equalsf(t, got, test.want["Before"], "pos.Before(%#v)", test.in)
			})
		}
	})
	t.Run("After", func(t *testing.T) {
		for i, test := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				got := pos.After(test.in)

				assert.Equalsf(t, got, test.want["After"], "pos.After(%#v)", test.in)
			})
		}
	})
}
