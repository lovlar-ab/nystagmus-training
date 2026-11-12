//go:build darwin

package main

import (
	"time"

	"github.com/ebitengine/purego/objc"

	"nystagmus-training/internal/training"
)

const (
	escapeKey = 53
	downKey   = 125
	upKey     = 126
	// Above menu bar and fullscreen apps.
	screenSaverLevel = 1000
	allSpaces        = uint(1<<0 | 1<<8)
	// Path samples per second; Core Animation interpolates.
	sampleRate = 30
	pathKey    = "path"
	hint       = "Follow the dot, head still. Esc stops. ↑↓ sets backdrop."
)

type overlay struct {
	window, dot, label, timer objc.ID
	running                   bool
}

func buildOverlay(frame NSRect) {
	o := &app.overlay
	o.window = cls("NTOverlayWindow").Send(sel("alloc")).Send(
		sel("initWithContentRect:styleMask:backing:defer:"), frame, uint(0), uint(2), false)
	o.window.Send(sel("setReleasedWhenClosed:"), false)
	o.window.Send(sel("setLevel:"), screenSaverLevel)
	o.window.Send(sel("setOpaque:"), false)
	o.window.Send(sel("setCollectionBehavior:"), allSpaces)
	content := o.window.Send(sel("contentView"))

	// A bare layer animates off-process.
	size := float64(training.DotSize)
	content.Send(sel("setWantsLayer:"), true)
	o.dot = cls("CALayer").Send(sel("layer"))
	o.dot.Send(sel("setBounds:"), rect(0, 0, size, size))
	o.dot.Send(sel("setCornerRadius:"), size/2)
	o.dot.Send(sel("setBackgroundColor:"), cls("NSColor").Send(sel("systemGreenColor")).Send(sel("CGColor")))
	content.Send(sel("layer")).Send(sel("addSublayer:"), o.dot)

	o.label = cls("NSTextField").Send(sel("labelWithString:"), nsString(hint))
	o.label.Send(sel("setTextColor:"), gray(0.55, 1))
	o.label.Send(sel("setFont:"), cls("NSFont").Send(sel("systemFontOfSize:"), float64(15)))
	o.label.Send(sel("sizeToFit"))
	content.Send(sel("addSubview:"), o.label)
}

func applyBackdrop() {
	alpha := float64(app.settings.Backdrop) / 100
	app.overlay.window.Send(sel("setBackgroundColor:"), gray(0, alpha))
}

// screenWidthCM returns 0 when unknown.
func screenWidthCM(screen objc.ID) float64 {
	number := screen.Send(sel("deviceDescription")).Send(sel("objectForKey:"), nsString("NSScreenNumber"))
	if number == 0 {
		return 0
	}
	return displaySize(objc.Send[uint32](number, sel("unsignedIntValue"))).Width / 10
}

func startSession() {
	o := &app.overlay
	if o.running {
		return
	}
	screen := cls("NSScreen").Send(sel("mainScreen"))
	if screen == 0 {
		// No display; usage stays due.
		return
	}
	frame := objc.Send[NSRect](screen, sel("frame"))
	if o.window == 0 {
		buildOverlay(frame)
	}
	applyBackdrop()
	o.window.Send(sel("setFrame:display:"), frame, true)
	width, height := frame.Size.Width, frame.Size.Height
	labelWidth := objc.Send[NSRect](o.label, sel("frame")).Size.Width
	o.label.Send(sel("setFrameOrigin:"), NSPoint{X: (width - labelWidth) / 2, Y: 24})
	// The start time seeds each path.
	path := training.NewPath(width, height, screenWidthCM(screen), uint64(time.Now().UnixNano()))
	length := app.settings.Length().Seconds()

	o.running = true
	app.ns.Send(sel("activateIgnoringOtherApps:"), true)
	o.window.Send(sel("makeKeyAndOrderFront:"), nilID)
	// Animate once the window is on screen.
	animateDot(path, width/2, height/2, length)
	cls("NSCursor").Send(sel("hide"))
	o.timer = startTimer(length, "finish:", false)
	refreshMenu()
}

func stopSession() {
	o := &app.overlay
	if !o.running {
		return
	}
	o.running = false
	// A firing timer lives through its callback.
	o.timer.Send(sel("invalidate"))
	o.timer = 0
	o.dot.Send(sel("removeAnimationForKey:"), nsString(pathKey))
	o.window.Send(sel("orderOut:"), nilID)
	cls("NSCursor").Send(sel("unhide"))
	// Hiding returns focus to the last app.
	app.ns.Send(sel("hide:"), nilID)
	app.usage.Reset()
	refreshMenu()
}

// animateDot hands the whole path to Core Animation.
func animateDot(path training.Path, centerX, centerY, seconds float64) {
	count := int(seconds*sampleRate) + 1
	points := make([]NSPoint, count)
	for i := range points {
		x, y := path.At(float64(i) / sampleRate)
		points[i] = NSPoint{X: centerX + x, Y: centerY + y}
	}
	// Even time steps match even keyframes.
	line := pathCreate()
	pathAddLines(line, 0, &points[0], uint(count))
	animation := cls("CAKeyframeAnimation").Send(sel("animationWithKeyPath:"), nsString("position"))
	animation.Send(sel("setPath:"), line)
	animation.Send(sel("setDuration:"), seconds)
	pathRelease(line)

	dot := app.overlay.dot
	dot.Send(sel("setPosition:"), points[0])
	dot.Send(sel("addAnimation:forKey:"), animation, nsString(pathKey))
}

func onFinish(_ objc.ID, _ objc.SEL, _ objc.ID) { stopSession() }

// onKey swallows all other keys.
func onKey(_ objc.ID, _ objc.SEL, event objc.ID) {
	switch objc.Send[uint16](event, sel("keyCode")) {
	case escapeKey:
		stopSession()
	case upKey:
		dimBackdrop(1)
	case downKey:
		dimBackdrop(-1)
	}
}

func dimBackdrop(steps int) {
	app.settings = app.settings.Dim(steps)
	applyBackdrop()
	saveSettings()
}
