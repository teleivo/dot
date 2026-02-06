package layout

import (
	"math"
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestSafeAdd(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tests := map[string]struct {
			a, b int
			want int
		}{
			"zero + zero":                     {0, 0, 0},
			"positive + positive":             {5, 3, 8},
			"negative + negative":             {-5, -3, -8},
			"positive + negative":             {5, -3, 2},
			"negative + positive":             {-5, 3, -2},
			"zero + MaxInt":                   {0, math.MaxInt, math.MaxInt},
			"MaxInt + zero":                   {math.MaxInt, 0, math.MaxInt},
			"edge of overflow":                {math.MaxInt - 1, 1, math.MaxInt},
			"edge of overflow (commutative)":  {1, math.MaxInt - 1, math.MaxInt},
			"zero + MinInt":                   {0, math.MinInt, math.MinInt},
			"MinInt + zero":                   {math.MinInt, 0, math.MinInt},
			"edge of underflow":               {math.MinInt + 1, -1, math.MinInt},
			"edge of underflow (commutative)": {-1, math.MinInt + 1, math.MinInt},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				assert.Equals(t, safeAdd(tt.a, tt.b), tt.want, "safeAdd(%d, %d)", tt.a, tt.b)
			})
		}
	})

	t.Run("Overflow", func(t *testing.T) {
		tests := map[string]struct {
			a, b int
		}{
			"MaxInt + 1":      {math.MaxInt, 1},
			"1 + MaxInt":      {1, math.MaxInt},
			"MaxInt + MaxInt": {math.MaxInt, math.MaxInt},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				defer func() {
					if err := recover(); err == nil {
						t.Errorf("safeAdd(%d, %d): want panic but got none", tt.a, tt.b)
					}
				}()
				_ = safeAdd(tt.a, tt.b)
			})
		}
	})

	t.Run("Underflow", func(t *testing.T) {
		tests := map[string]struct {
			a, b int
		}{
			"MinInt + -1":     {math.MinInt, -1},
			"-1 + MinInt":     {-1, math.MinInt},
			"MinInt + MinInt": {math.MinInt, math.MinInt},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				defer func() {
					if err := recover(); err == nil {
						t.Errorf("safeAdd(%d, %d): want panic but got none", tt.a, tt.b)
					}
				}()
				_ = safeAdd(tt.a, tt.b)
			})
		}
	})
}
