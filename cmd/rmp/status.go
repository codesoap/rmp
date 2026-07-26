package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/codesoap/rmp/internal/library"
	"github.com/codesoap/rmp/internal/playercom"
)

func handlePlayerStatus(state *uiState, status playercom.Status) {
	switch status.Type {
	case playercom.StatusError:
		state.seeking = false
		state.error = status.Err.Error()
	case playercom.StatusPlayingNext:
		state.seeking = false
		state.currentlyPlaying += 1
		if state.currentlyPlaying >= len(state.queue) {
			state.statusLine = "error"
			break
		}
		if state.queue[state.currentlyPlaying].ID != status.SongID {
			// Something is wrong, this is not the song that should be playing.
			state.statusLine = "error"
			break
		}
		queueStatusLoading(state)
		setNextIfPresent(state)
	case playercom.StatusPosition:
		if !state.seeking {
			state.position = status.Position
			setStatusPlaying(state, state.position)
		}
	case playercom.StatusStopped:
		state.seeking = false
		state.currentlyPlaying = -1
		state.statusLine = "stopped"
	case playercom.StatusNowPlaying:
		go func() {
			err := library.Scrobble(state.cfg, status.SongID, false)
			if err != nil {
				err = fmt.Errorf("could not register now playing: %s", err)
				state.events <- event{etype: eventError, err: err}
			}
		}()
	case playercom.StatusScrobble:
		if state.noScrobble {
			break
		}
		go func() {
			err := library.Scrobble(state.cfg, status.SongID, true)
			if err != nil {
				err = fmt.Errorf("could not scrobble: %s", err)
				state.events <- event{etype: eventError, err: err}
			}
		}()
	}
	draw(*state)
}

// queueStatusLoading will show a loading status, if playback does not
// start within 100ms.
func queueStatusLoading(state *uiState) {
	state.loadingStatusDebouncer = time.AfterFunc(100*time.Millisecond, func() {
		state.events <- event{etype: eventLoadingSong}
	})
}

func setStatusPlaying(state *uiState, pos time.Duration) {
	if state.currentlyPlaying < 0 || state.currentlyPlaying >= len(state.queue) {
		state.statusLine = "error"
		return
	}
	seconds := int(pos / time.Second)
	dur := state.queue[state.currentlyPlaying].Duration
	state.statusLine = fmt.Sprintf("playing at %02d:%02d / %02d:%02d",
		seconds/60, seconds%60, dur/60, dur%60)
}

func setStatusPaused(state *uiState) {
	state.statusLine = strings.Replace(state.statusLine, "playing", "paused", 1)
}

func setStatusResumed(state *uiState) {
	state.statusLine = strings.Replace(state.statusLine, "paused", "playing", 1)
}
