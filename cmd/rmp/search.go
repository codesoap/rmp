package main

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/clipperhouse/displaywidth"
	"github.com/codesoap/rmp/internal/library"
	"github.com/codesoap/rmp/internal/song"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

func startSearch(state *uiState) {
	state.searchShown = true
	state.searchFilter = ""
	state.searchOptions = state.allSongs
	updateSearchMatches(state)
}

func startSimilarSearch(state *uiState, songID string) {
	state.loadingSimilarDebouncer = time.AfterFunc(100*time.Millisecond, func() {
		state.events <- event{etype: eventLoadingSimilar}
	})
	ctx, cancel := context.WithCancel(context.Background())
	state.cancelLoadingSimilar = cancel
	go func() {
		songs, err := library.FetchSimilarSongs(ctx, state.cfg, songID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				err = nil // The user triggered this, so don't show an error.
			}
			state.events <- event{etype: eventError, err: err}
			return
		}
		// Not synchronized, but no one should be touching searchOptions at
		// this time:
		state.searchOptions = songs
		state.events <- event{etype: eventLoadedSimilar}
	}()
}

func updateSearchMatches(state *uiState) {
	state.searchSelection = 0
	clear(state.searchPicks)
	state.searchMatches = fuzzySorted(state.searchFilter, state.searchOptions)
}

func fuzzySorted(pattern string, songs []song.Song) []song.Song {
	type rankedSong struct {
		song  song.Song
		score int
	}

	slab := util.MakeSlab(100*1024, 2048)
	patternRunes := []rune(pattern)

	caseInsensitiveSearch := pattern == strings.ToLower(pattern)

	results := make([]rankedSong, 0, len(songs))
	for _, song := range songs {
		str := song.String()
		if caseInsensitiveSearch {
			str = strings.ToLower(str)
		}
		chars := util.RunesToChars([]rune(str))
		r, _ := algo.FuzzyMatchV2(false, true, true, &chars, patternRunes, false, slab)
		if r.Start >= 0 {
			results = append(results, rankedSong{song: song, score: r.Score})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].song.String() < results[j].song.String()
	})

	ret := make([]song.Song, len(results))
	for i, rs := range results {
		ret[i] = rs.song
	}
	return ret
}

func toggleSearchSelection(state *uiState) {
	if _, found := state.searchPicks[state.searchSelection]; found {
		delete(state.searchPicks, state.searchSelection)
	} else {
		state.searchPicks[state.searchSelection] = struct{}{}
	}
}

func drawSearch(state uiState) {
	w, h := state.s.Size()
	boxW := min(w-2, 122)
	boxH := h - 2
	x0 := (w - boxW) / 2
	y0 := (h - boxH) / 2
	x1 := x0 + boxW - 1
	y1 := y0 + boxH - 1
	drawSearchBox(state.s, x0, y0, x1, y1)

	if state.searchFilter == "" {
		help := "enter fuzzy search... (queue with enter, multi select with tab)"
		style := tcell.StyleDefault.Foreground(color.DimGray)
		searchLine := displaywidth.TruncateString(help, boxW-4, "")
		state.s.PutStrStyled(x0+2, y0+1, searchLine, style)
		style = tcell.StyleDefault.Reverse(true)
		state.s.PutStrStyled(x0+2, y0+1, "e", style)
	} else {
		// FIXME: Show end of line when longer than input field.
		searchLine := displaywidth.TruncateString(state.searchFilter, boxW-5, "")
		state.s.PutStr(x0+2, y0+1, searchLine)
		style := tcell.StyleDefault.Reverse(true)
		state.s.PutStrStyled(x0+2+displaywidth.String(searchLine), y0+1, " ", style)
	}

	from := 0
	selectionH := boxH - 4
	if selectionH < len(state.searchMatches) {
		from = max(0, state.searchSelection-selectionH/2)
		from = min(from, len(state.searchMatches)-selectionH)
	}
	for i, song := range state.searchMatches[from:min(from+selectionH, len(state.searchMatches))] {
		if _, pick := state.searchPicks[from+i]; pick {
			state.s.PutStr(x0+2, y0+3+i, "▌")
		}
		l := displaywidth.TruncateString(song.String(), boxW-5, "")
		if i+from == state.searchSelection {
			state.s.PutStrStyled(x0+3, y0+3+i, l, tcell.StyleDefault.Reverse(true))
		} else {
			state.s.PutStr(x0+3, y0+3+i, l)
		}
	}
}
