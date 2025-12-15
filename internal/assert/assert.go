// Package assert provides runtime assertion checking for invariants.
package assert

import "fmt"

// That panics if condition is false.
func That(condition bool, msg string, args ...any) {
	if condition {
		return
	}

	if len(args) > 0 {
		panic(fmt.Sprintf(msg, args...))
	}
	panic(msg)
}
