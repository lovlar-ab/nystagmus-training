package training

import (
	"math"
	"testing"
	"time"
)

func TestUsageCountsActiveTime(t *testing.T) {
	const every = 2 * time.Hour
	var u Usage
	u.Add(Tick, 0)
	u.Add(Tick, IdleLimit) // away
	u.Add(8*time.Hour, 0)  // woke from sleep
	u.Add(-time.Second, 0) // clock moved back
	if want := 3 * Tick; every-u.Left(every) != want {
		t.Fatalf("active = %v, want %v", every-u.Left(every), want)
	}
}

func TestUsageDueAfterInterval(t *testing.T) {
	for _, every := range []time.Duration{15 * time.Minute, 2 * time.Hour, 4 * time.Hour} {
		var u Usage
		for i := 0; i < int(every/Tick)-1; i++ {
			u.Add(Tick, 0)
		}
		if u.Due(every) {
			t.Fatalf("%v: due one tick early", every)
		}
		u.Add(Tick, 0)
		if !u.Due(every) || u.Left(every) != 0 {
			t.Fatalf("not due at %v", every)
		}
		u.Reset()
		if u.Due(every) || u.Left(every) != every {
			t.Fatalf("%v: reset did not clear usage", every)
		}
	}
}

func TestShorterIntervalIsDueAtOnce(t *testing.T) {
	var u Usage
	for i := 0; i < int(time.Hour/Tick); i++ {
		u.Add(Tick, 0)
	}
	if u.Due(2*time.Hour) || !u.Due(30*time.Minute) {
		t.Fatal("1 h of use: want due for 30 min only")
	}
}

func TestLeftText(t *testing.T) {
	cases := map[time.Duration]string{
		0:                              "0 min",
		time.Second:                    "1 min",
		52 * time.Minute:               "52 min",
		time.Hour:                      "1 h",
		time.Hour + 51*time.Minute + 1: "1 h 52 min",
		2 * time.Hour:                  "2 h",
	}
	for d, want := range cases {
		if got := LeftText(d); got != want {
			t.Errorf("LeftText(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestChecked(t *testing.T) {
	cases := []struct{ in, want Settings }{
		{Settings{}, defaults}, // nothing saved
		{Settings{-5, 16, 101, Window{600, 630}}, defaults},
		{Settings{20, 5, 90, Window{485, 1080}}, defaults}, // off step
		{Settings{15, 15, 10, Window{0, 1440}}, Settings{15, 15, 10, Window{0, 1440}}},
		{Settings{240, 1, 100, Window{540, 600}}, Settings{240, 1, 100, Window{540, 600}}},
	}
	for _, c := range cases {
		if got := c.in.Checked(); got != c.want {
			t.Errorf("%+v.Checked() = %+v, want %+v", c.in, got, c.want)
		}
	}
	if defaults.Every() != 2*time.Hour || defaults.Length() != 5*time.Minute {
		t.Errorf("defaults = %v every, %v long", defaults.Every(), defaults.Length())
	}
}

func TestDimClamps(t *testing.T) {
	s := defaults
	if got := s.Dim(1).Backdrop; got != 100 {
		t.Errorf("90 up = %d, want 100", got)
	}
	if got := s.Dim(5).Backdrop; got != MaxBackdrop {
		t.Errorf("up past max = %d", got)
	}
	if got := s.Dim(-1).Backdrop; got != 80 {
		t.Errorf("90 down = %d, want 80", got)
	}
	if got := s.Dim(-20).Backdrop; got != MinBackdrop {
		t.Errorf("down past min = %d", got)
	}
}

func TestScaleSnap(t *testing.T) {
	cases := []struct {
		scale    Scale
		fraction float64
		want     int
	}{
		{EveryScale, -0.3, 15},
		{EveryScale, 0, 15},
		{EveryScale, 1, 240},
		{EveryScale, 7, 240},
		{EveryScale, 105.0 / 225, 120},
		{EveryScale, 0.49, 120}, // 125 min rounds down
		{LengthScale, 4.0 / 14, 5},
		{DayScale, 0.5, 720},
		{DayScale, 0.51, 720},
		{DayScale, 0.52, 750},
	}
	for _, c := range cases {
		got := c.scale.Snap(c.fraction)
		if got != c.want || !c.scale.Has(got) {
			t.Errorf("%+v.Snap(%v) = %d, want %d", c.scale, c.fraction, got, c.want)
		}
	}
	for _, s := range []Scale{EveryScale, LengthScale, DayScale} {
		for v := s.Min; v <= s.Max; v += s.Step {
			if got := s.Snap(s.Fraction(v)); got != v {
				t.Errorf("%+v: %d round-trips to %d", s, v, got)
			}
		}
	}
}

func TestNearest(t *testing.T) {
	window := []int{480, 1080}
	for target, want := range map[int]int{0: 0, 480: 0, 779: 0, 781: 1, 1440: 1} {
		if got := Nearest(window, target); got != want {
			t.Errorf("Nearest(%v, %d) = %d, want %d", window, target, got, want)
		}
	}
	if got := Nearest([]int{120}, 15); got != 0 {
		t.Errorf("single handle = %d, want 0", got)
	}
}

func TestWindow(t *testing.T) {
	w := Window{480, 1080}
	for minute, want := range map[int]bool{0: false, 479: false, 480: true, 1079: true, 1080: false} {
		if got := w.Contains(minute); got != want {
			t.Errorf("Contains(%d) = %v, want %v", minute, got, want)
		}
	}
	if !(Window{0, 1440}).Contains(1439) {
		t.Error("full day excludes 23:59")
	}
	drags := []struct {
		start  bool
		minute int
		want   Window
	}{
		{true, 540, Window{540, 1080}},
		{true, 1440, Window{1020, 1080}}, // start stops before end
		{false, 1440, Window{480, 1440}},
		{false, 0, Window{480, 540}}, // end stops after start
	}
	for _, d := range drags {
		if got := w.Drag(d.start, d.minute); got != d.want || !got.valid() {
			t.Errorf("Drag(%v, %d) = %+v, want %+v", d.start, d.minute, got, d.want)
		}
	}
	if got := (Window{0, 1440}).Text(); got != "00:00–24:00" {
		t.Errorf("Text = %q", got)
	}
	if got := w.Text(); got != "08:00–18:00" {
		t.Errorf("Text = %q", got)
	}
}

func TestPathStartsAtCenter(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		x, y := NewPath(1470, 956, 28.6, seed).At(0)
		if x != 0 || y != 0 {
			t.Fatalf("seed %d: start = %v,%v, want 0,0", seed, x, y)
		}
	}
}

func TestPathDiffersBySeed(t *testing.T) {
	const at = 60.0
	x1, y1 := NewPath(1470, 956, 28.6, 1).At(at)
	x2, y2 := NewPath(1470, 956, 28.6, 2).At(at)
	if math.Hypot(x1-x2, y1-y2) < 1 {
		t.Fatalf("seeds 1 and 2 give the same point %v,%v", x1, y1)
	}
	x3, y3 := NewPath(1470, 956, 28.6, 1).At(at)
	if x1 != x3 || y1 != y3 {
		t.Fatal("same seed gave a different path")
	}
}

func TestPathStaysOnScreen(t *testing.T) {
	const w, h = 1470.0, 956.0
	longest := float64(LengthScale.Max * 60)
	for seed := uint64(0); seed < 20; seed++ {
		p := NewPath(w, h, 28.6, seed)
		for s := 0.0; s < longest; s += 0.05 {
			x, y := p.At(s)
			if math.Abs(x) > w/2-DotSize/2 || math.Abs(y) > h/2-DotSize/2 {
				t.Fatalf("seed %d: dot off screen at %vs: %v,%v", seed, s, x, y)
			}
		}
	}
}

func TestPathPeakSpeed(t *testing.T) {
	// 13 inch and 27 inch screens.
	for _, screen := range [][3]float64{{1470, 956, 28.6}, {2560, 1440, 59.7}} {
		for seed := uint64(0); seed < 5; seed++ {
			w, cm := screen[0], screen[2]
			p := NewPath(w, screen[1], cm, seed)
			degPerPoint := cm / w / (ViewDistance * math.Tan(math.Pi/180))
			peak := 0.0
			const dt = 0.01
			for s := RampTime; s < defaults.Length().Seconds(); s += dt {
				x0, _ := p.At(s)
				x1, _ := p.At(s + dt)
				peak = math.Max(peak, math.Abs(x1-x0)/dt*degPerPoint)
			}
			if math.Abs(peak-PeakSpeed) > 0.05 {
				t.Errorf("%v cm, seed %d: peak = %.2f deg/s, want %v", cm, seed, peak, PeakSpeed)
			}
		}
	}
}

func TestPathWithoutScreenSize(t *testing.T) {
	p := NewPath(1470, 956, 0, 1)
	if p.fx != fallbackFrequency {
		t.Fatalf("fx = %v, want %v", p.fx, fallbackFrequency)
	}
}
