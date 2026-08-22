package main

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/codesoap/rmp/internal/config"
	"github.com/codesoap/rmp/internal/song"
	"github.com/gdamore/tcell/v3"
)

type uiState struct {
	cfg config.Config

	s      tcell.Screen
	events chan event

	helpShown bool

	allSongs                []song.Song
	searchShown             bool
	searchSelection         int // Index of selected entry in searchMatches.
	searchOptions           []song.Song
	searchFilter            string
	searchMatches           []song.Song
	searchPicks             map[int]struct{} // Keys are indexes.
	loadingSimilarDebouncer *time.Timer
	loadingSimilar          bool
	cancelLoadingSimilar    context.CancelFunc

	statusLine             string
	loadingStatusDebouncer *time.Timer

	error string

	paused           bool
	position         time.Duration // Position in the currently playing song; guessed when seeking.
	seeking          bool          // Seek in progress.
	queue            []song.Song
	queueSelection   int // Index of the selected entry in queue.
	currentlyPlaying int // Index of the currently playing entry in queue.

	noScrobble bool
	singleMode bool
}

func launchUI(cfg config.Config, allSongs []song.Song) {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Init(); err != nil {
		log.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			s.Fini()
			log.Fatalf("panic: %v\n%s\n", r, debug.Stack())
		}
	}()

	state := uiState{
		cfg:              cfg,
		s:                s,
		events:           make(chan event),
		allSongs:         allSongs,
		searchPicks:      make(map[int]struct{}),
		statusLine:       "stopped",
		currentlyPlaying: -1,
	}
	draw(state)
	runEventLoop(&state)
}

func runEventLoop(state *uiState) {
	for {
		select {
		case ev, ok := <-state.s.EventQ():
			if !ok {
				return
			}
			handleTcellEvent(state, ev)
		case status := <-p.Status():
			if state.loadingStatusDebouncer != nil {
				state.loadingStatusDebouncer.Stop()
			}
			handlePlayerStatus(state, status)
		case ev := <-state.events:
			switch ev.etype {
			case eventLoadingSong:
				state.statusLine = "loading..."
			case eventLoadingSimilar:
				if !state.searchShown {
					// The condition ensures that the loading didn't already complete.
					state.loadingSimilar = true
				}
			case eventLoadedSimilar:
				state.loadingSimilar = false
				state.searchShown = true
				state.searchFilter = ""
				updateSearchMatches(state)
			case eventError:
				if state.loadingSimilarDebouncer != nil {
					state.loadingSimilarDebouncer.Stop()
				}
				state.loadingSimilar = false
				if ev.err != nil {
					state.error = ev.err.Error()
				}
			}
			draw(*state)
		case fb := <-p.PlayerFeedback():
			if fb.CmdID != playerCmdID {
				break
			}
			if fb.Err != nil {
				state.error = fmt.Sprintf("Player command: %s", fb.Err)
				draw(*state)
			} else {
				state.seeking = false
				draw(*state)
			}
		case fb := <-p.QueueFeedback():
			if fb.Err != nil && fb.CmdID == queueCmdID {
				state.error = fmt.Sprintf("Queue command: %s", fb.Err)
				draw(*state)
			}
		}
	}
}

func handleTcellEvent(state *uiState, ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		state.s.Sync()
		draw(*state)
	case *tcell.EventKey:
		if state.helpShown {
			state.helpShown = false
			draw(*state)
		}
		switch {
		case state.error != "":
			handleErrorKey(ev, state)
		case state.loadingSimilar:
			handleLoadingSimilarKey(ev, state)
		case state.searchShown:
			handleSearchKey(ev, state)
		default:
			handleKey(ev, state)
		}
	}
}
