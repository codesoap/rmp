package main

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

func draw(state uiState) {
	state.s.Clear()
	w, h := state.s.Size()
	if w < 12 || h < 8 {
		state.s.PutStr(0, 0, "Screen too small")
		state.s.Show()
		return
	}
	if state.seeking {
		style := tcell.StyleDefault.Foreground(color.DimGray)
		state.s.PutStrStyled(0, 0, state.statusLine, style)
	} else {
		state.s.PutStr(0, 0, state.statusLine)
	}
	drawModes(state)
	drawQueue(state)
	if len(state.queue) == 0 {
		x := max(0, w/2-11)
		y := max(0, h/2-1)
		style := tcell.StyleDefault.Foreground(color.DimGray)
		state.s.PutStrStyled(x, y, "Press q to quit.", style)
		state.s.PutStrStyled(x, y+1, "Press / to queue songs.", style)
		state.s.PutStrStyled(x, y+2, "Press ? for more info.", style)

	}
	if state.helpShown {
		drawHelp(state)
	}
	if state.searchShown {
		drawSearch(state)
	}
	if state.loadingSimilar {
		w, h := state.s.Size()
		text := "Loading similar songs..."
		boxW := len(text) + 4 // padding
		boxH := 3
		x0 := (w - boxW) / 2
		y0 := (h - boxH) / 2
		x1 := x0 + boxW - 1
		y1 := y0 + boxH - 1
		drawBox(state.s, x0, y0, x1, y1)
		state.s.PutStr(x0+2, y0+1, text)
	}
	if state.error != "" {
		text := "Error: " + state.error
		text = displaywidth.TruncateString(text, w-6, "…")
		boxW := len(text) + 4 // padding
		boxH := 3
		x0 := (w - boxW) / 2
		y0 := (h - boxH) / 2
		x1 := x0 + boxW - 1
		y1 := y0 + boxH
		drawBox(state.s, x0, y0, x1, y1)
		style := tcell.StyleDefault.Foreground(color.Red)
		state.s.PutStrStyled(x0+2, y0+1, text, style)
		style = tcell.StyleDefault.Foreground(color.DimGray)
		state.s.PutStrStyled(x0+2, y0+2, "Press esc to close.", style)
	}
	state.s.Show()
}

func drawModes(state uiState) {
	var modes []string
	if state.singleMode {
		modes = append(modes, "single")
	}
	if state.noScrobble {
		modes = append(modes, "scrobbling off")
	}
	if len(modes) > 0 {
		s := strings.Join(modes, ", ")
		dw := displaywidth.String(s)
		w, _ := state.s.Size()
		state.s.PutStr(max(0, w-dw-2), 0, " "+s)
	}
}
