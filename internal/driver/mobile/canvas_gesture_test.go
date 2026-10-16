//go:build mobile && (!windows || !ci)

package mobile

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	intdriver "fyne.io/fyne/v2/internal/driver"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"testing"
)

func gestureTap(c *canvas, pos fyne.Position) {
	c.tapUp(pos, 0, func(w fyne.Tappable, e *fyne.PointEvent) { w.Tapped(e) },
		func(w fyne.SecondaryTappable, e *fyne.PointEvent) { w.TappedSecondary(e) },
		func(w fyne.DoubleTappable, e *fyne.PointEvent) { w.DoubleTapped(e) },
		func(w fyne.Draggable, e *fyne.DragEvent) { w.Dragged(e) })
}

func Test_canvas_GestureKeyboardResizeKeepsTarget(t *testing.T) {
	d.DoFromGoroutine(func() {
		c := newCanvas(fyne.CurrentDevice()).(*canvas)
		c.OnFocus = nil
		entry := widget.NewEntry()
		firstCalls, secondCalls := 0, 0
		first := widget.NewButton("Test", func() { firstCalls++ })
		second := widget.NewButton("Delete", func() { secondCalls++ })
		c.SetContent(container.NewWithoutLayout(entry, first, second))
		c.Resize(fyne.NewSize(240, 240))
		entry.Resize(fyne.NewSize(160, 35))
		first.Resize(fyne.NewSize(160, 40))
		second.Resize(fyne.NewSize(160, 40))
		first.Move(fyne.NewPos(0, 60))
		second.Move(fyne.NewPos(0, 120))
		c.Focus(entry)
		c.OnUnfocus = func() {
			first.Move(fyne.NewPos(0, 120))
			second.Move(fyne.NewPos(0, 60))
		}
		pos := fyne.NewPos(20, 80)
		c.tapDown(pos, 0)
		assert.Same(t, entry, c.Focused(), "finger-down must not dismiss the keyboard")
		gestureTap(c, pos)
		assert.Equal(t, 1, firstCalls, "the original button must receive the first tap")
		assert.Zero(t, secondCalls, "keyboard resize must never activate another button")
	}, true)
}

func Test_canvas_GestureCancelsChangedTarget(t *testing.T) {
	for _, change := range []string{"move", "hide", "disable", "remove", "overlay", "swipe"} {
		t.Run(change, func(t *testing.T) {
			d.DoFromGoroutine(func() {
				c := newCanvas(fyne.CurrentDevice()).(*canvas)
				c.OnFocus, c.OnUnfocus = nil, nil
				calls := 0
				first := widget.NewButton("Test", func() { calls++ })
				second := widget.NewButton("Other", func() { calls++ })
				content := container.NewWithoutLayout(first, second)
				c.SetContent(content)
				c.Resize(fyne.NewSize(240, 240))
				first.Resize(fyne.NewSize(160, 40))
				second.Resize(fyne.NewSize(160, 40))
				first.Move(fyne.NewPos(0, 60))
				second.Move(fyne.NewPos(0, 120))
				pos := fyne.NewPos(20, 80)
				c.tapDown(pos, 0)
				switch change {
				case "move":
					first.Move(fyne.NewPos(0, 120))
					second.Move(fyne.NewPos(0, 60))
				case "hide":
					first.Hide()
				case "disable":
					first.Disable()
				case "remove":
					content.Remove(first)
				case "overlay":
					popup := widget.NewModalPopUp(widget.NewButton("Overlay", func() { calls++ }), c)
					popup.Show()
				case "swipe":
					c.tapMove(fyne.NewPos(45, 80), 0, func(w fyne.Draggable, e *fyne.DragEvent) { w.Dragged(e) })
				}
				gestureTap(c, pos)
				assert.Zero(t, calls, "a cancelled or retargeted gesture must not activate a control")
			}, true)
		})
	}
}

func Test_canvas_GestureSwipeOverOtherEntryKeepsFocus(t *testing.T) {
	d.DoFromGoroutine(func() {
		w := d.CreateWindow("Gesture test").(*window)
		defer w.Close()
		c := w.canvas
		c.OnFocus, c.OnUnfocus = nil, nil
		first, second := widget.NewEntry(), widget.NewEntry()
		content := container.NewVBox(first, second)
		for i := 0; i < 15; i++ {
			content.Add(widget.NewLabel("Form row"))
		}
		scroll := container.NewVScroll(content)
		c.SetContent(scroll)
		c.Resize(fyne.NewSize(300, 200))
		c.Focus(first)
		pos := intdriver.AbsolutePositionForObject(second, c.ObjectTrees()).AddXY(20, 15)
		c.tapDown(pos, 0)
		assert.Same(t, first, c.Focused(), "touching another field is not yet a tap")
		c.tapMove(pos.AddXY(0, -20), 0, func(w fyne.Draggable, e *fyne.DragEvent) { w.Dragged(e) })
		gestureTap(c, pos.AddXY(0, -20))
		assert.Same(t, first, c.Focused())
		assert.Greater(t, scroll.Offset.Y, float32(0), "the form must scroll over an entry")
	}, true)
}

func Test_canvas_GestureModalResizeKeepsTarget(t *testing.T) {
	d.DoFromGoroutine(func() {
		c := newCanvas(fyne.CurrentDevice()).(*canvas)
		c.OnFocus = nil
		c.SetContent(widget.NewLabel("Background"))
		c.Resize(fyne.NewSize(360, 260))
		calls, otherCalls := 0, 0
		entry := widget.NewEntry()
		button := widget.NewButton("Test", func() { calls++ })
		other := widget.NewButton("Other", func() { otherCalls++ })
		popup := widget.NewModalPopUp(container.NewVBox(entry, button, other), c)
		popup.Show()
		c.Focus(entry)
		c.OnUnfocus = func() { c.Resize(fyne.NewSize(360, 600)) }
		pos := intdriver.AbsolutePositionForObject(button, c.ObjectTrees()).AddXY(15, button.Size().Height/2)
		c.tapDown(pos, 0)
		assert.True(t, c.Focused() == entry)
		gestureTap(c, pos)
		assert.Equal(t, 1, calls)
		assert.Zero(t, otherCalls)
		popup.Hide()
	}, true)
}

func Test_canvas_GestureTapTransfersEntryFocus(t *testing.T) {
	d.DoFromGoroutine(func() {
		w := d.CreateWindow("Tap focus").(*window)
		defer w.Close()
		c := w.canvas
		hidden := 0
		c.OnFocus = nil
		c.OnUnfocus = func() { hidden++ }
		first, second := widget.NewEntry(), widget.NewPasswordEntry()
		c.SetContent(container.NewVBox(first, second))
		c.Resize(fyne.NewSize(300, 200))
		c.Focus(first)
		pos := intdriver.AbsolutePositionForObject(second, c.ObjectTrees()).AddXY(20, 15)
		c.tapDown(pos, 0)
		assert.True(t, c.Focused() == first)
		gestureTap(c, pos)
		assert.True(t, c.Focused() == second)
		assert.Zero(t, hidden, "field-to-field taps must not hide the keyboard")
	}, true)
}

func Test_canvas_GestureKeyboardResizeRevealsFocusedEntry(t *testing.T) {
	d.DoFromGoroutine(func() {
		c := newCanvas(fyne.CurrentDevice()).(*canvas)
		c.OnFocus, c.OnUnfocus = nil, nil
		entry := widget.NewEntry()
		content := container.NewVBox()
		for i := 0; i < 12; i++ {
			content.Add(widget.NewLabel("Form row"))
		}
		content.Add(entry)
		scroll := container.NewVScroll(content)
		c.SetContent(scroll)
		c.Resize(fyne.NewSize(360, 600))
		c.Focus(entry)
		c.Resize(fyne.NewSize(360, 180))
		assert.Greater(t, scroll.Offset.Y, float32(0))
		top := intdriver.AbsolutePositionForObject(entry, c.ObjectTrees()).Y
		viewport := intdriver.AbsolutePositionForObject(scroll, c.ObjectTrees()).Y
		assert.GreaterOrEqual(t, top, viewport)
		assert.LessOrEqual(t, top+entry.Size().Height, viewport+scroll.Size().Height)
		scroll.ScrollToTop()
		assert.Zero(t, scroll.Offset.Y, "manual scrolling must remain possible with focus")
	}, true)
}

func Test_canvas_GestureWindowHeaderTarget(t *testing.T) {
	for _, focused := range []bool{false, true} {
		t.Run(fmt.Sprint("focused=", focused), func(t *testing.T) {
			d.DoFromGoroutine(func() {
				c := newCanvas(fyne.CurrentDevice()).(*canvas)
				c.OnFocus, c.OnUnfocus = nil, nil
				entry := widget.NewEntry()
				c.SetContent(entry)
				calls := 0
				button := widget.NewButton("Close", func() { calls++ })
				c.setWindowHead(container.NewWithoutLayout(button))
				c.Resize(fyne.NewSize(320, 480))
				c.windowHead.Move(fyne.NewPos(0, 0))
				c.windowHead.Resize(fyne.NewSize(320, 48))
				button.Move(fyne.NewPos(200, 0))
				button.Resize(fyne.NewSize(100, 40))
				if focused {
					c.Focus(entry)
				}
				pos := fyne.NewPos(220, 20)
				target, _, _ := c.findTapTarget(pos)
				assert.Same(t, button, target)
				c.tapDown(pos, 0)
				gestureTap(c, pos)
				assert.Equal(t, 1, calls, "window header buttons must receive valid taps")
				button.Hide()
				assert.False(t, c.tapTargetActive(button, nil))
				button.Show()
				c.setWindowHead(container.NewWithoutLayout())
				assert.False(t, c.tapTargetActive(button, nil), "replaced headers must not receive stale taps")
			}, true)
		})
	}
}
