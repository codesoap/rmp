package main

import (
	"github.com/gdamore/tcell/v3"
)

func drawQueue(state uiState) {
	_, h := state.s.Size()
	from := 0
	queueH := h - 1 // Top line is status, not queue.
	if queueH < len(state.queue) {
		from = max(0, state.queueSelection-queueH/2)
		from = min(from, len(state.queue)-queueH)
	}
	for i, song := range state.queue[from:min(from+queueH, len(state.queue))] {
		if i+from == state.currentlyPlaying {
			state.s.PutStr(0, i+1, ">")
		}
		if i+from == state.queueSelection {
			state.s.PutStrStyled(2, i+1, song.String(), tcell.StyleDefault.Reverse(true))
		} else {
			state.s.PutStr(2, i+1, song.String())
		}
	}
}

func handleQueueUp(alt bool, state *uiState) {
	if alt {
		s := state.queueSelection
		if s > 0 {
			changedNext := false
			switch s {
			case state.currentlyPlaying:
				state.currentlyPlaying -= 1
				changedNext = true
			case state.currentlyPlaying + 1:
				changedNext = true
				state.currentlyPlaying += 1
			case state.currentlyPlaying + 2:
				changedNext = true
			}
			state.queue[s], state.queue[s-1] = state.queue[s-1], state.queue[s]
			if changedNext {
				setNextIfPresent(state)
			}
		}
	}
	state.queueSelection = max(state.queueSelection-1, 0)
	draw(*state)
}

func handleQueueDown(alt bool, state *uiState) {
	if alt {
		s := state.queueSelection
		if s < len(state.queue)-1 {
			changedNext := false
			switch s {
			case state.currentlyPlaying - 1:
				state.currentlyPlaying -= 1
				changedNext = true
			case state.currentlyPlaying:
				state.currentlyPlaying += 1
				changedNext = true
			case state.currentlyPlaying + 1:
				changedNext = true
			}
			state.queue[s], state.queue[s+1] = state.queue[s+1], state.queue[s]
			if changedNext {
				setNextIfPresent(state)
			}
		}
	}
	state.queueSelection = max(0, min(state.queueSelection+1, len(state.queue)-1))
	draw(*state)
}
