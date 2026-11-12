// Package training holds the session rules.
package training

import (
	"fmt"
	"time"
)

const (
	// Longer input gaps count as away.
	IdleLimit = time.Minute
	Tick      = 5 * time.Second
)

// Usage counts active computer time.
type Usage struct{ active time.Duration }

// Add counts elapsed time when active.
func (u *Usage) Add(elapsed, idle time.Duration) {
	if idle >= IdleLimit || elapsed <= 0 {
		return
	}
	// Sleep stretches the wall clock.
	if elapsed > 2*Tick {
		elapsed = 2 * Tick
	}
	u.active += elapsed
}

func (u *Usage) Due(every time.Duration) bool { return u.active >= every }

func (u *Usage) Left(every time.Duration) time.Duration {
	if u.Due(every) {
		return 0
	}
	return every - u.active
}

func (u *Usage) Reset() { u.active = 0 }

// LeftText rounds up to whole minutes.
func LeftText(d time.Duration) string {
	minutes := int((d + time.Minute - 1) / time.Minute)
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%d h", minutes/60)
	}
	return fmt.Sprintf("%d h %d min", minutes/60, minutes%60)
}
