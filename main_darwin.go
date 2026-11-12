//go:build darwin

package main

import (
	"os"
	"runtime"
	"time"

	"github.com/ebitengine/purego/objc"

	"nystagmus-training/internal/login"
	"nystagmus-training/internal/training"
)

const (
	accessoryPolicy    = 1
	variableLength     = float64(-1)
	combinedState      = int32(0)
	anyInputEvent      = ^uint32(0)
	loginFailedMessage = "Open at login did not change"
	// NSUserDefaults keys.
	everyKey    = "everyMinutes"
	lengthKey   = "lengthMinutes"
	backdropKey = "backdropPercent"
	startKey    = "windowStartMinute"
	endKey      = "windowEndMinute"
)

var app struct {
	ns, controller                     objc.ID
	statusLine, sessionItem, loginItem objc.ID
	sliders                            []*slider
	settings                           training.Settings
	usage                              training.Usage
	lastTick                           time.Time
	overlay                            overlay
}

func init() {
	// AppKit needs the main thread.
	runtime.LockOSThread()
}

func main() {
	loadFrameworks()
	app.ns = cls("NSApplication").Send(sel("sharedApplication"))
	app.ns.Send(sel("setActivationPolicy:"), accessoryPolicy)
	registerClasses()
	app.controller = cls("NTController").Send(sel("new"))
	loadSettings()
	buildMenu()
	if err := syncLogin(); err != nil {
		showAlert(loginFailedMessage, err.Error())
	}
	app.lastTick = time.Now()
	startTimer(training.Tick.Seconds(), "tick:", true)
	app.ns.Send(sel("run"))
}

func defaults() objc.ID { return cls("NSUserDefaults").Send(sel("standardUserDefaults")) }

// loadSettings reads 0 for unsaved keys.
func loadSettings() {
	read := func(key string) int {
		return objc.Send[int](defaults(), sel("integerForKey:"), nsString(key))
	}
	app.settings = training.Settings{
		EveryMinutes:  read(everyKey),
		LengthMinutes: read(lengthKey),
		Backdrop:      read(backdropKey),
		Window:        training.Window{Start: read(startKey), End: read(endKey)},
	}.Checked()
}

func saveSettings() {
	write := func(key string, value int) {
		defaults().Send(sel("setInteger:forKey:"), value, nsString(key))
	}
	s := app.settings
	write(everyKey, s.EveryMinutes)
	write(lengthKey, s.LengthMinutes)
	write(backdropKey, s.Backdrop)
	write(startKey, s.Window.Start)
	write(endKey, s.Window.End)
}

func buildMenu() {
	item := cls("NSStatusBar").Send(sel("systemStatusBar")).Send(sel("statusItemWithLength:"), variableLength)
	item.Send(sel("retain"))
	button := item.Send(sel("button"))
	icon := cls("NSImage").Send(sel("imageWithSystemSymbolName:accessibilityDescription:"),
		nsString("eye"), nsString("Nystagmus training"))
	if icon != 0 {
		button.Send(sel("setImage:"), icon)
	} else {
		button.Send(sel("setTitle:"), nsString("◉"))
	}

	menu := cls("NSMenu").Send(sel("alloc")).Send(sel("init"))
	// No action keeps the line disabled.
	app.statusLine = addItem(menu, "", 0, nilID, "")
	app.sessionItem = addItem(menu, "", sel("toggleSession:"), app.controller, "")
	menu.Send(sel("addItem:"), cls("NSMenuItem").Send(sel("separatorItem")))
	app.sliders = settingSliders()
	for _, s := range app.sliders {
		addSlider(menu, s)
	}
	menu.Send(sel("addItem:"), cls("NSMenuItem").Send(sel("separatorItem")))
	app.loginItem = addItem(menu, "Open at login", sel("toggleLogin:"), app.controller, "")
	addItem(menu, "Quit", sel("terminate:"), app.ns, "q")
	item.Send(sel("setMenu:"), menu)
	refreshMenu()
}

func addItem(menu objc.ID, title string, action objc.SEL, target objc.ID, key string) objc.ID {
	item := menu.Send(sel("addItemWithTitle:action:keyEquivalent:"), nsString(title), action, nsString(key))
	item.Send(sel("setTarget:"), target)
	return item
}

func settingSliders() []*slider {
	set := &app.settings
	return []*slider{
		{
			scale: training.DayScale,
			read:  func() []int { return []int{set.Window.Start, set.Window.End} },
			write: func(handle, value int) { set.Window = set.Window.Drag(handle == 0, value) },
			title: func() string { return "Active " + set.Window.Text() },
		},
		{
			scale: training.EveryScale,
			read:  func() []int { return []int{set.EveryMinutes} },
			write: func(_, value int) { set.EveryMinutes = value },
			title: func() string { return "Start after " + training.LeftText(set.Every()) + " of use" },
		},
		{
			scale: training.LengthScale,
			read:  func() []int { return []int{set.LengthMinutes} },
			write: func(_, value int) { set.LengthMinutes = value },
			title: func() string { return "Session length " + training.LeftText(set.Length()) },
		},
	}
}

// minuteOfDay is local clock time.
func minuteOfDay(now time.Time) int { return now.Hour()*60 + now.Minute() }

func refreshMenu() {
	s := app.settings
	status, session := "Next session after "+training.LeftText(app.usage.Left(s.Every()))+" of use", "Start session"
	switch {
	case app.overlay.running:
		status, session = "Session running", "Stop session"
	case !s.Window.Contains(minuteOfDay(time.Now())):
		status = "Paused until " + training.ClockText(s.Window.Start)
	}
	app.statusLine.Send(sel("setTitle:"), nsString(status))
	app.sessionItem.Send(sel("setTitle:"), nsString(session))
	for _, slider := range app.sliders {
		slider.layout()
	}
	state := 0
	if agent, err := login.Default(); err == nil && agent.Enabled() {
		state = 1
	}
	app.loginItem.Send(sel("setState:"), state)
}

func onTick(_ objc.ID, _ objc.SEL, _ objc.ID) {
	now := time.Now()
	elapsed := now.Sub(app.lastTick)
	app.lastTick = now
	if app.overlay.running {
		return
	}
	idle := time.Duration(idleSeconds(combinedState, anyInputEvent) * float64(time.Second))
	app.usage.Add(elapsed, idle)
	// A due session waits for the window.
	if app.usage.Due(app.settings.Every()) && app.settings.Window.Contains(minuteOfDay(now)) {
		startSession()
	}
	refreshMenu()
}

func onSession(_ objc.ID, _ objc.SEL, _ objc.ID) {
	if app.overlay.running {
		stopSession()
	} else {
		startSession()
	}
}

func onLogin(_ objc.ID, _ objc.SEL, _ objc.ID) {
	if err := toggleLogin(); err != nil {
		showAlert(loginFailedMessage, err.Error())
	}
	refreshMenu()
}

func toggleLogin() error {
	agent, err := login.Default()
	if err != nil {
		return err
	}
	if agent.Enabled() {
		return agent.Disable()
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return agent.Enable(exe)
}

// syncLogin follows the app when moved.
func syncLogin() error {
	agent, err := login.Default()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return agent.Sync(exe)
}
