package main

import (
	"github.com/codesoap/rmp/internal/playercom"
	"github.com/gdamore/tcell/v3"
)

func drawBox(s tcell.Screen, x0, y0, x1, y1 int) {
	for x := x0 + 1; x <= x1; x++ {
		s.SetContent(x, y0, tcell.RuneHLine, nil, tcell.StyleDefault)
		s.SetContent(x, y1, tcell.RuneHLine, nil, tcell.StyleDefault)
	}
	for y := y0 + 1; y <= y1; y++ {
		s.SetContent(x0, y, tcell.RuneVLine, nil, tcell.StyleDefault)
		s.SetContent(x1, y, tcell.RuneVLine, nil, tcell.StyleDefault)
	}
	s.SetContent(x0, y0, tcell.RuneULCorner, nil, tcell.StyleDefault)
	s.SetContent(x1, y0, tcell.RuneURCorner, nil, tcell.StyleDefault)
	s.SetContent(x0, y1, tcell.RuneLLCorner, nil, tcell.StyleDefault)
	s.SetContent(x1, y1, tcell.RuneLRCorner, nil, tcell.StyleDefault)

	for y := y0 + 1; y < y1; y++ {
		for x := x0 + 1; x < x1; x++ {
			s.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}
}

func drawSearchBox(s tcell.Screen, x0, y0, x1, y1 int) {
	for x := x0 + 1; x < x1; x++ {
		s.SetContent(x, y0, tcell.RuneHLine, nil, tcell.StyleDefault)
		s.SetContent(x, y1, tcell.RuneHLine, nil, tcell.StyleDefault)
		s.SetContent(x, y0+2, tcell.RuneHLine, nil, tcell.StyleDefault)
	}
	for y := y0 + 1; y < y1; y++ {
		if y == y0+2 {
			s.SetContent(x0, y, tcell.RuneLTee, nil, tcell.StyleDefault)
			s.SetContent(x1, y, tcell.RuneRTee, nil, tcell.StyleDefault)
		} else {
			s.SetContent(x0, y, tcell.RuneVLine, nil, tcell.StyleDefault)
			s.SetContent(x1, y, tcell.RuneVLine, nil, tcell.StyleDefault)
		}
	}
	s.SetContent(x0, y0, tcell.RuneULCorner, nil, tcell.StyleDefault)
	s.SetContent(x1, y0, tcell.RuneURCorner, nil, tcell.StyleDefault)
	s.SetContent(x0, y1, tcell.RuneLLCorner, nil, tcell.StyleDefault)
	s.SetContent(x1, y1, tcell.RuneLRCorner, nil, tcell.StyleDefault)

	for y := y0 + 1; y < y1; y++ {
		if y == y0+2 {
			continue
		}
		for x := x0 + 1; x < x1; x++ {
			s.SetContent(x, y, ' ', nil, tcell.StyleDefault)
		}
	}
}

func setNextIfPresent(state *uiState) bool {
	if state.singleMode {
		return false
	}
	queueCmdID++
	if state.currentlyPlaying >= 0 && len(state.queue) > state.currentlyPlaying+1 {
		select {
		case p.QueueCmds() <- playercom.QueueCmd{
			CmdID: queueCmdID,
			Op:    playercom.QueueSetNext,
			Song:  state.queue[state.currentlyPlaying+1],
		}:
		default:
			state.error = "Too many commands. Slow down."
			return false
		}
	} else {
		select {
		case p.QueueCmds() <- playercom.QueueCmd{
			CmdID: queueCmdID,
			Op:    playercom.QueueRemoveNext,
		}:
		default:
			state.error = "Too many commands. Slow down."
			return false
		}
	}
	return true
}
