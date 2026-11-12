//go:build darwin

package main

import (
	"fmt"
	"structs"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

type NSPoint struct {
	_    structs.HostLayout
	X, Y float64
}

type NSSize struct {
	_             structs.HostLayout
	Width, Height float64
}

type NSRect struct {
	_      structs.HostLayout
	Origin NSPoint
	Size   NSSize
}

const nilID = objc.ID(0)

func rect(x, y, width, height float64) NSRect {
	return NSRect{Origin: NSPoint{X: x, Y: y}, Size: NSSize{Width: width, Height: height}}
}

var (
	// Seconds since last input event.
	idleSeconds func(state int32, event uint32) float64
	// Physical display size, millimeters.
	displaySize func(display uint32) NSSize
)

func loadFrameworks() {
	const flags = purego.RTLD_GLOBAL | purego.RTLD_LAZY
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", flags); err != nil {
		panic(fmt.Errorf("load AppKit: %w", err))
	}
	graphics, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", flags)
	if err != nil {
		panic(fmt.Errorf("load CoreGraphics: %w", err))
	}
	purego.RegisterLibFunc(&idleSeconds, graphics, "CGEventSourceSecondsSinceLastEventType")
	purego.RegisterLibFunc(&displaySize, graphics, "CGDisplayScreenSize")
}

func sel(name string) objc.SEL { return objc.RegisterName(name) }

func cls(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

func nsString(text string) objc.ID {
	return cls("NSString").Send(sel("stringWithUTF8String:"), text)
}

func startTimer(seconds float64, action string) objc.ID {
	return cls("NSTimer").Send(
		sel("scheduledTimerWithTimeInterval:target:selector:userInfo:repeats:"),
		seconds, app.controller, sel(action), nilID, true)
}

func showAlert(message, detail string) {
	app.ns.Send(sel("activateIgnoringOtherApps:"), true)
	alert := cls("NSAlert").Send(sel("alloc")).Send(sel("init"))
	alert.Send(sel("setMessageText:"), nsString(message))
	alert.Send(sel("setInformativeText:"), nsString(detail))
	alert.Send(sel("runModal"))
	alert.Send(sel("release"))
}

func registerClasses() {
	_, err := objc.RegisterClass("NTController", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
		{Cmd: sel("tick:"), Fn: onTick},
		{Cmd: sel("frame:"), Fn: onFrame},
		{Cmd: sel("toggleSession:"), Fn: onSession},
		{Cmd: sel("toggleLogin:"), Fn: onLogin},
	})
	if err != nil {
		panic(fmt.Errorf("register NTController: %w", err))
	}
	// Subviews pass mouse events up here.
	_, err = objc.RegisterClass("NTSliderView", objc.GetClass("NSView"), nil, nil, []objc.MethodDef{
		{Cmd: sel("mouseDown:"), Fn: onSliderDown},
		{Cmd: sel("mouseDragged:"), Fn: onSliderDrag},
	})
	if err != nil {
		panic(fmt.Errorf("register NTSliderView: %w", err))
	}
	_, err = objc.RegisterClass("NTOverlayWindow", objc.GetClass("NSWindow"), nil, nil, []objc.MethodDef{
		// Borderless windows refuse keys by default.
		{Cmd: sel("canBecomeKeyWindow"), Fn: func(objc.ID, objc.SEL) bool { return true }},
		{Cmd: sel("keyDown:"), Fn: onKey},
	})
	if err != nil {
		panic(fmt.Errorf("register NTOverlayWindow: %w", err))
	}
}
