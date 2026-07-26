package main

import (
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/codesoap/rmp/internal/playercom"
	"github.com/codesoap/rmp/internal/song"
	"github.com/gdamore/tcell/v3"
)

var playerCmdID int
var queueCmdID int

func handleErrorKey(ev *tcell.EventKey, state *uiState) {
	switch ev.Key() {
	case tcell.KeyEscape:
		state.error = ""
		draw(*state)
	}
}

func handleHelpKey(ev *tcell.EventKey, state *uiState) {
	switch ev.Key() {
	case tcell.KeyEscape:
		state.helpShown = false
		state.helpScroll = 0
		draw(*state)
	case tcell.KeyUp:
		state.helpScroll = max(state.helpScroll-1, 0)
		draw(*state)
	case tcell.KeyDown:
		_, h := state.s.Size()
		availableHeight := h - 4 // One padding and border line on each side.
		maxScroll := max(len(helpText)+3-availableHeight, 0)
		state.helpScroll = min(state.helpScroll+1, maxScroll)
		draw(*state)
	}
}

func handleLoadingSimilarKey(ev *tcell.EventKey, state *uiState) {
	switch ev.Key() {
	case tcell.KeyEscape:
		if state.cancelLoadingSimilar != nil {
			state.cancelLoadingSimilar()
		}
		state.loadingSimilar = false
		draw(*state)
	}
}

func handleSearchKey(ev *tcell.EventKey, state *uiState) {
	switch ev.Key() {
	case tcell.KeyEscape:
		state.searchShown = false
		state.searchFilter = ""
		draw(*state)
	case tcell.KeyUp:
		state.searchSelection = max(state.searchSelection-1, 0)
		draw(*state)
	case tcell.KeyPgUp:
		_, h := state.s.Size()
		shownSongCnt := h - 6
		state.searchSelection = max(state.searchSelection-shownSongCnt, 0)
		draw(*state)
	case tcell.KeyDown:
		state.searchSelection = max(0, min(state.searchSelection+1, len(state.searchMatches)-1))
		draw(*state)
	case tcell.KeyPgDn:
		_, h := state.s.Size()
		shownSongCnt := h - 6
		state.searchSelection = max(0, min(state.searchSelection+shownSongCnt, len(state.searchMatches)-1))
		draw(*state)
	case tcell.KeyRune:
		state.searchFilter += ev.Str()
		updateSearchMatches(state)
		draw(*state)
	case tcell.KeyBackspace:
		runes := []rune(state.searchFilter)
		state.searchFilter = string(runes[:max(0, len(runes)-1)])
		updateSearchMatches(state)
		draw(*state)
	case tcell.KeyCtrlW:
		i := strings.LastIndexFunc(state.searchFilter, unicode.IsSpace)
		if i > -1 {
			state.searchFilter = strings.TrimRightFunc(state.searchFilter[:i], unicode.IsSpace)
		} else {
			state.searchFilter = ""
		}
		updateSearchMatches(state)
		draw(*state)
	case tcell.KeyTab:
		toggleSearchSelection(state)
		state.searchSelection = min(state.searchSelection+1, len(state.searchMatches)-1)
		draw(*state)
	case tcell.KeyEnter:
		state.searchShown = false
		state.searchFilter = ""
		wasPlayingLastSong := len(state.queue) > 0 && state.currentlyPlaying+1 == len(state.queue)
		if len(state.searchPicks) == 0 {
			if len(state.searchMatches) > state.searchSelection {
				state.queue = append(state.queue, state.searchMatches[state.searchSelection])
			}
			state.queueSelection = len(state.queue) - 1 // Select added song.
		} else {
			picks := make([]int, 0, len(state.queue))
			for pick := range state.searchPicks {
				picks = append(picks, pick)
			}
			slices.Sort(picks)
			state.queueSelection = len(state.queue) // Select first added song.
			for _, pick := range picks {
				state.queue = append(state.queue, state.searchMatches[pick])
			}
		}
		if wasPlayingLastSong {
			setNextIfPresent(state)
		}
		draw(*state)
	}
}

func handleKey(ev *tcell.EventKey, state *uiState) {
	switch ev.Key() {
	case tcell.KeyUp:
		handleQueueUp(ev.Modifiers()&tcell.ModAlt != 0, state)
	case tcell.KeyDown:
		handleQueueDown(ev.Modifiers()&tcell.ModAlt != 0, state)
	case tcell.KeyEnter:
		if state.queueSelection < len(state.queue) {
			state.currentlyPlaying = state.queueSelection
			playerCmdID++
			select {
			case p.PlayerCmds() <- playercom.PlayerCmd{
				CmdID: playerCmdID,
				Op:    playercom.PlayerPlay,
				Song:  state.queue[state.queueSelection],
			}:
				queueStatusLoading(state)
				state.paused = false
				// This can cause loading two songs in parallel, but I think it's OK:
				setNextIfPresent(state)
			default:
				state.error = "Too many commands. Slow down."
				draw(*state)
			}
		}
	case tcell.KeyLeft:
		if state.currentlyPlaying >= 0 && state.currentlyPlaying < len(state.queue) {
			state.position = max(0, state.position-10*time.Second)
			playerCmdID++
			select {
			case p.PlayerCmds() <- playercom.PlayerCmd{
				CmdID:  playerCmdID,
				Op:     playercom.PlayerSeek,
				Song:   state.queue[state.currentlyPlaying],
				SeekTo: state.position,
			}:
				state.seeking = true
				setStatusPlaying(state, state.position)
				if state.paused {
					setStatusPaused(state)
				}
				draw(*state)
			default:
				state.error = "Too many commands. Slow down."
			}
		}
	case tcell.KeyRight:
		if state.currentlyPlaying >= 0 && state.currentlyPlaying < len(state.queue) {
			dur := time.Duration(state.queue[state.currentlyPlaying].Duration) * time.Second
			state.position = min(dur, state.position+10*time.Second)
			playerCmdID++
			select {
			case p.PlayerCmds() <- playercom.PlayerCmd{
				CmdID:  playerCmdID,
				Op:     playercom.PlayerSeek,
				Song:   state.queue[state.currentlyPlaying],
				SeekTo: state.position,
			}:
				state.seeking = true
				setStatusPlaying(state, state.position)
				if state.paused {
					setStatusPaused(state)
				}
				draw(*state)
			default:
				state.error = "Too many commands. Slow down."
			}
		}
	case tcell.KeyRune:
		switch ev.Str() {
		case "q":
			state.s.Fini()
		case "?":
			state.helpShown = true
			draw(*state)
		case "/":
			startSearch(state)
			draw(*state)
		case "s":
			if len(state.queue) > 0 && state.queueSelection < len(state.queue) {
				startSimilarSearch(state, state.queue[state.queueSelection].ID)
				draw(*state)
			}
		case "k":
			handleQueueUp(ev.Modifiers()&tcell.ModAlt != 0, state)
		case "j":
			handleQueueDown(ev.Modifiers()&tcell.ModAlt != 0, state)
		case "d":
			s := state.queueSelection
			if s < len(state.queue) {
				state.queue = append(state.queue[:s], state.queue[s+1:]...)
			}
			if s == len(state.queue) {
				state.queueSelection = max(0, state.queueSelection-1)
			}
			switch {
			case s < state.currentlyPlaying:
				state.currentlyPlaying -= 1
			case s == state.currentlyPlaying:
				playerCmdID++
				select {
				case p.PlayerCmds() <- playercom.PlayerCmd{
					CmdID: playerCmdID,
					Op:    playercom.PlayerStop,
				}:
					state.currentlyPlaying = -1
				default:
					state.error = "Too many commands. Slow down."
				}
			case s == state.currentlyPlaying+1:
				setNextIfPresent(state)
			}
			draw(*state)
		case "C":
			if state.currentlyPlaying >= 0 && state.currentlyPlaying < len(state.queue) {
				queueCmdID++
				select {
				case p.QueueCmds() <- playercom.QueueCmd{
					CmdID: queueCmdID,
					Op:    playercom.QueueRemoveNext,
				}:
					state.queue = []song.Song{state.queue[state.currentlyPlaying]}
					state.currentlyPlaying = 0
					state.queueSelection = 0
				default:
					state.error = "Too many commands. Slow down."
				}
				draw(*state)
				break
			}
			fallthrough // If there is no playing song, clear all.
		case "c":
			playerCmdID++
			select {
			case p.PlayerCmds() <- playercom.PlayerCmd{
				CmdID: playerCmdID,
				Op:    playercom.PlayerStop,
			}:
				state.queue = state.queue[:0]
				state.queueSelection = 0
			default:
				state.error = "Too many commands. Slow down."
			}
			draw(*state)
		case " ":
			playerCmdID++
			var op playercom.PlayerOp
			if state.paused {
				op = playercom.PlayerResume
				state.paused = false
			} else {
				op = playercom.PlayerPause
				state.paused = true
			}
			select {
			case p.PlayerCmds() <- playercom.PlayerCmd{CmdID: playerCmdID, Op: op}:
				if state.paused {
					setStatusPaused(state)
				} else {
					setStatusResumed(state)
				}
			default:
				state.error = "Too many commands. Slow down."
			}
			draw(*state)
		case "S":
			if state.singleMode {
				if state.currentlyPlaying >= 0 && len(state.queue) > state.currentlyPlaying+1 {
					queueCmdID++
					select {
					case p.QueueCmds() <- playercom.QueueCmd{
						CmdID: queueCmdID,
						Op:    playercom.QueueSetNext,
						Song:  state.queue[state.currentlyPlaying+1],
					}:
						state.singleMode = false
					default:
						state.error = "Too many commands. Slow down."
					}
				} else {
					state.singleMode = false
				}
			} else {
				queueCmdID++
				select {
				case p.QueueCmds() <- playercom.QueueCmd{
					CmdID: queueCmdID,
					Op:    playercom.QueueRemoveNext,
				}:
					state.singleMode = true
				default:
					state.error = "Too many commands. Slow down."
				}
			}
			draw(*state)
		case "N":
			if state.noScrobble {
				state.noScrobble = false
			} else {
				state.noScrobble = true
			}
			draw(*state)
		}
	}
}
