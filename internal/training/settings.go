package training

import (
	"fmt"
	"math"
	"time"
)

const (
	DefaultBackdrop = 90
	// Backdrop opacity bounds and step, percent.
	MinBackdrop  = 10
	MaxBackdrop  = 100
	BackdropStep = 10
	// Shortest daily window, minutes.
	MinWindow = 60
)

// Scale is one slider's range.
type Scale struct{ Min, Max, Step int }

var (
	// Active use before a session, minutes.
	EveryScale = Scale{15, 240, 15}
	// Session duration, minutes.
	LengthScale = Scale{1, 15, 1}
	// Clock time, minutes since midnight.
	DayScale = Scale{0, 24 * 60, 30}
)

var defaults = Settings{
	EveryMinutes:  120,
	LengthMinutes: 5,
	Backdrop:      DefaultBackdrop,
	Window:        Window{Start: 8 * 60, End: 18 * 60},
}

// Snap maps a track fraction to a value.
func (s Scale) Snap(fraction float64) int {
	fraction = math.Min(1, math.Max(0, fraction))
	steps := math.Round(fraction * float64(s.Max-s.Min) / float64(s.Step))
	return s.Min + int(steps)*s.Step
}

// Fraction is the value's track position.
func (s Scale) Fraction(value int) float64 {
	return float64(value-s.Min) / float64(s.Max-s.Min)
}

func (s Scale) Has(value int) bool {
	return value >= s.Min && value <= s.Max && (value-s.Min)%s.Step == 0
}

// Nearest returns the closest value's index.
func Nearest(values []int, target int) int {
	best := 0
	for i, v := range values {
		if abs(v-target) < abs(values[best]-target) {
			best = i
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Window is the daily active clock range.
type Window struct{ Start, End int }

// Contains is true from Start until End.
func (w Window) Contains(minute int) bool { return minute >= w.Start && minute < w.End }

func (w Window) valid() bool {
	return DayScale.Has(w.Start) && DayScale.Has(w.End) && w.End-w.Start >= MinWindow
}

// Drag moves one end, keeping MinWindow.
func (w Window) Drag(start bool, minute int) Window {
	if start {
		w.Start = min(minute, w.End-MinWindow)
	} else {
		w.End = max(minute, w.Start+MinWindow)
	}
	return w
}

func (w Window) Text() string { return ClockText(w.Start) + "–" + ClockText(w.End) }

func ClockText(minute int) string { return fmt.Sprintf("%02d:%02d", minute/60, minute%60) }

// Settings are the user's saved choices.
type Settings struct {
	// Active use before a session.
	EveryMinutes int
	// Session duration.
	LengthMinutes int
	// Backdrop opacity, percent.
	Backdrop int
	// Sessions start only inside Window.
	Window Window
}

// Checked replaces invalid values with defaults.
func (s Settings) Checked() Settings {
	if !EveryScale.Has(s.EveryMinutes) {
		s.EveryMinutes = defaults.EveryMinutes
	}
	if !LengthScale.Has(s.LengthMinutes) {
		s.LengthMinutes = defaults.LengthMinutes
	}
	if s.Backdrop < MinBackdrop || s.Backdrop > MaxBackdrop {
		s.Backdrop = defaults.Backdrop
	}
	if !s.Window.valid() {
		s.Window = defaults.Window
	}
	return s
}

func (s Settings) Every() time.Duration { return time.Duration(s.EveryMinutes) * time.Minute }

func (s Settings) Length() time.Duration { return time.Duration(s.LengthMinutes) * time.Minute }

// Dim moves the backdrop by whole steps.
func (s Settings) Dim(steps int) Settings {
	s.Backdrop = min(MaxBackdrop, max(MinBackdrop, s.Backdrop+steps*BackdropStep))
	return s
}
