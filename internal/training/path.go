package training

import (
	"math"
	"math/rand/v2"
)

const (
	// Peak horizontal dot speed, deg/s.
	PeakSpeed = 8.0
	// Assumed eye to screen distance, cm.
	ViewDistance = 60.0
	// Seconds until full amplitude.
	RampTime = 10.0
	// Dot diameter, points.
	DotSize = 28.0
	// Horizontal frequency without screen size, Hz.
	fallbackFrequency = 0.1
	// Vertical to horizontal frequency ratio range.
	minRatio, maxRatio = 0.6, 0.85
)

// Path is a smooth dot path.
type Path struct{ ax, ay, fx, fy, phaseX, phaseY float64 }

// NewPath fits a seeded path to one screen.
func NewPath(width, height, widthCM float64, seed uint64) Path {
	ax, ay := 0.42*width, 0.40*height
	fx := fallbackFrequency
	if widthCM > 0 && width > 0 {
		cmPerDegree := ViewDistance * math.Tan(math.Pi/180)
		degrees := ax / width * widthCM / cmPerDegree
		fx = PeakSpeed / (2 * math.Pi * degrees)
	}
	random := rand.New(rand.NewPCG(seed, seed))
	ratio := minRatio + (maxRatio-minRatio)*random.Float64()
	return Path{
		ax: ax, ay: ay, fx: fx, fy: fx * ratio,
		phaseX: 2 * math.Pi * random.Float64(),
		phaseY: 2 * math.Pi * random.Float64(),
	}
}

// At returns the offset from screen center.
func (p Path) At(seconds float64) (x, y float64) {
	// The ramp starts every path at center.
	gain := ramp(seconds / RampTime)
	x = gain * p.ax * math.Sin(2*math.Pi*p.fx*seconds+p.phaseX)
	y = gain * p.ay * math.Sin(2*math.Pi*p.fy*seconds+p.phaseY)
	return x, y
}

// ramp is smoothstep from 0 to 1.
func ramp(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}
