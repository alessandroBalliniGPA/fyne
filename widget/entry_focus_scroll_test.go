package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestEntry_FocusDoesNotRevealCursor(t *testing.T) {
	test.NewApp()
	e := NewMultiLineEntry()
	e.SetText(strings.Repeat("one line of text\n", 30))
	e.Validator = func(string) error { return nil }
	e.Resize(fyne.NewSize(220, 80))
	test.WidgetRenderer(e)
	for _, focus := range []struct {
		name   string
		change func()
	}{
		{"gained", e.FocusGained}, {"lost", e.FocusLost},
	} {
		t.Run(focus.name, func(t *testing.T) {
			e.scroller.ScrollToOffset(fyne.NewPos(0, 150))
			offset := e.scroller.Offset
			focus.change()
			assert.Equal(t, offset, e.scroller.Offset, "focus changes must not undo manual scrolling")
		})
	}
	e.FocusGained()
	e.scroller.ScrollToOffset(fyne.NewPos(0, 150))
	e.TypedRune('x')
	assert.Less(t, e.scroller.Offset.Y, float32(150), "typing must still reveal the cursor")
}
