//go:build darwin

package main

import (
	"github.com/ebitengine/purego/objc"

	"nystagmus-training/internal/training"
)

// Slider row geometry, points.
const (
	sliderWidth  = 280.0
	sliderHeight = 48.0
	sliderInset  = 20.0
	knobSize     = 16.0
	trackHeight  = 4.0
	trackY       = 12.0
	trackLeft    = sliderInset + knobSize/2
	trackWidth   = sliderWidth - 2*trackLeft
)

// slider is one menu row of handles.
type slider struct {
	view, label, fill objc.ID
	handles           []objc.ID
	scale             training.Scale
	// read returns one value per handle.
	read  func() []int
	write func(handle, value int)
	title func() string
	// Handle held during a drag.
	grabbed int
}

// sliders finds a row by its view.
var sliders = map[objc.ID]*slider{}

func gray(white, alpha float64) objc.ID {
	return cls("NSColor").Send(sel("colorWithWhite:alpha:"), white, alpha)
}

// layerView adds a rounded, filled view.
func layerView(parent objc.ID, frame NSRect, color objc.ID, radius float64) objc.ID {
	view := cls("NSView").Send(sel("alloc")).Send(sel("initWithFrame:"), frame)
	view.Send(sel("setWantsLayer:"), true)
	layer := view.Send(sel("layer"))
	layer.Send(sel("setBackgroundColor:"), color.Send(sel("CGColor")))
	layer.Send(sel("setCornerRadius:"), radius)
	parent.Send(sel("addSubview:"), view)
	return view
}

func addSlider(menu objc.ID, s *slider) {
	s.view = cls("NTSliderView").Send(sel("alloc")).Send(sel("initWithFrame:"),
		rect(0, 0, sliderWidth, sliderHeight))
	s.label = cls("NSTextField").Send(sel("labelWithString:"), nsString(""))
	s.label.Send(sel("setFont:"), cls("NSFont").Send(sel("menuFontOfSize:"), float64(0)))
	s.label.Send(sel("setFrame:"), rect(sliderInset, 26, sliderWidth-2*sliderInset, 18))
	s.view.Send(sel("addSubview:"), s.label)

	track := rect(trackLeft, trackY, trackWidth, trackHeight)
	layerView(s.view, track, gray(0.5, 0.4), trackHeight/2)
	s.fill = layerView(s.view, track, cls("NSColor").Send(sel("systemGreenColor")), trackHeight/2)
	for range s.read() {
		handle := layerView(s.view, rect(0, 0, knobSize, knobSize), gray(1, 1), knobSize/2)
		layer := handle.Send(sel("layer"))
		layer.Send(sel("setBorderWidth:"), float64(1))
		layer.Send(sel("setBorderColor:"), gray(0.5, 0.6).Send(sel("CGColor")))
		s.handles = append(s.handles, handle)
	}

	addItem(menu, "", 0, nilID, "").Send(sel("setView:"), s.view)
	sliders[s.view] = s
	s.layout()
}

func (s *slider) position(value int) float64 {
	return trackLeft + s.scale.Fraction(value)*trackWidth
}

// layout places handles, fill and label.
func (s *slider) layout() {
	values := s.read()
	for i, handle := range s.handles {
		handle.Send(sel("setFrameOrigin:"),
			NSPoint{X: s.position(values[i]) - knobSize/2, Y: trackY + trackHeight/2 - knobSize/2})
	}
	// One handle fills from the left.
	from, to := trackLeft, s.position(values[len(values)-1])
	if len(values) > 1 {
		from = s.position(values[0])
	}
	s.fill.Send(sel("setFrame:"), rect(from, trackY, to-from, trackHeight))
	s.label.Send(sel("setStringValue:"), nsString(s.title()))
}

func (s *slider) valueAt(event objc.ID) int {
	inWindow := objc.Send[NSPoint](event, sel("locationInWindow"))
	local := objc.Send[NSPoint](s.view, sel("convertPoint:fromView:"), inWindow, nilID)
	return s.scale.Snap((local.X - trackLeft) / trackWidth)
}

func (s *slider) drag(event objc.ID) {
	s.write(s.grabbed, s.valueAt(event))
	app.settings = app.settings.Checked()
	saveSettings()
	refreshMenu()
}

func onSliderDown(view objc.ID, _ objc.SEL, event objc.ID) {
	s := sliders[view]
	if s == nil {
		return
	}
	s.grabbed = training.Nearest(s.read(), s.valueAt(event))
	s.drag(event)
}

func onSliderDrag(view objc.ID, _ objc.SEL, event objc.ID) {
	if s := sliders[view]; s != nil {
		s.drag(event)
	}
}
