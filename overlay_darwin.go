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
	hint             = "Follow the dot, head still. Esc stops. ↑↓ sets backdrop."
)

type overlay struct {
	window, dot, label, timer objc.ID
	path                      training.Path
	centerX, centerY          float64
	started                   time.Time
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

	size := float64(training.DotSize)
	o.dot = layerView(content, rect(0, 0, size, size), cls("NSColor").Send(sel("systemGreenColor")), size/2)

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
	o.started = time.Now()
	// The start time seeds each path.
	seed := uint64(o.started.UnixNano())
	o.path = training.NewPath(width, height, screenWidthCM(screen), seed)
	o.centerX, o.centerY = width/2, height/2
	labelWidth := objc.Send[NSRect](o.label, sel("frame")).Size.Width
	o.label.Send(sel("setFrameOrigin:"), NSPoint{X: (width - labelWidth) / 2, Y: 24})
	moveDot(0)

	o.running = true
	app.ns.Send(sel("activateIgnoringOtherApps:"), true)
	o.window.Send(sel("makeKeyAndOrderFront:"), nilID)
	cls("NSCursor").Send(sel("hide"))
	o.timer = startTimer(1.0/60, "frame:")
	refreshMenu()
}

func stopSession() {
	o := &app.overlay
	if !o.running {
		return
	}
	o.running = false
	o.timer.Send(sel("invalidate"))
	o.timer = 0
	o.window.Send(sel("orderOut:"), nilID)
	cls("NSCursor").Send(sel("unhide"))
	// Hiding returns focus to the last app.
	app.ns.Send(sel("hide:"), nilID)
	app.usage.Reset()
	refreshMenu()
}

func moveDot(seconds float64) {
	o := &app.overlay
	x, y := o.path.At(seconds)
	radius := float64(training.DotSize) / 2
	o.dot.Send(sel("setFrameOrigin:"), NSPoint{X: o.centerX + x - radius, Y: o.centerY + y - radius})
}

func onFrame(_ objc.ID, _ objc.SEL, _ objc.ID) {
	o := &app.overlay
	if !o.running {
		return
	}
	elapsed := time.Since(o.started)
	if elapsed >= app.settings.Length() {
		stopSession()
		return
	}
	moveDot(elapsed.Seconds())
}

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
