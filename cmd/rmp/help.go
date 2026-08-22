package main

import (
	"fmt"

	"github.com/clipperhouse/displaywidth"
	"github.com/codesoap/rmp/internal/library"
)

var helpText = []string{
	"Commands in the queue:",
	"?    : show this help",
	"q    : quit",
	"/    : search whole library",
	"s    : search similar songs to current selection",
	"enter: play selected song",
	"space: toggle pause/play",
	"→    : seek forwards 10s",
	"←    : seek backwards 10s",
	"↑,k  : select song above",
	"↓,j  : select song below",
	"K,alt+↑,k: move selected song up",
	"J,alt+↓,j: move selected song down",
	"d    : delete song from queue",
	"c    : clear queue",
	"C    : clear queue except currently playing",
	"S    : toggle single mode (stop after each song)",
	"N    : toggle scrobbling",
	"",
	"Commands in search dialogs:",
	"enter: append selected song to queue",
	"↑    : select song above",
	"↓    : select song below",
	"PgUp : go one screen up",
	"PgDn : go one screen down",
	"tab  : select song; only needed for multiple selections",
	"esc  : abort",
	"",
	"Command in error popups:",
	"esc  : close",
}

func drawHelp(state uiState) {
	cacheInfo1 := fmt.Sprintf("Songs are cached at '%s'.\n", library.CacheDir)
	cacheInfo2 := "When no song is playing, you may delete them."
	longHelpText := append(helpText, "", cacheInfo1, cacheInfo2)

	w, h := state.s.Size()
	boxW := min(w-2, max(64, displaywidth.String(cacheInfo1)))
	boxH := min(h-2, len(longHelpText)+2)
	x0 := (w - boxW) / 2
	y0 := (h - boxH) / 2
	x1 := x0 + boxW - 1
	y1 := y0 + boxH - 1
	drawBox(state.s, x0, y0, x1, y1)
	var i int
	var line string
	for i, line = range longHelpText {
		if y0+i+2 > y1 {
			break
		}
		line = displaywidth.TruncateString(line, boxW-4, "")
		state.s.PutStr(x0+2, y0+1+i, line)
	}
}
