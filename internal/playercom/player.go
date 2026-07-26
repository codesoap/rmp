package playercom

import (
	"time"

	"github.com/codesoap/rmp/internal/song"
)

// PlayerCmd is a command that can be sent to the player. If a new
// command is given before the previous finished, it is assumed that the
// previous will be canceled.
type PlayerCmd struct {
	CmdID  int
	Op     PlayerOp
	Song   song.Song     // Only relevant for Op == PlayerPlay and Op == PlayerSeek.
	SeekTo time.Duration // Only relevant for Op == PlayerSeek.
}

// PlayerFeedback is the feedback the player gives to a given PlayerCmd.
// The ID will identify which PlayerCmd is being responded to. Every
// PlayerCmd will get PlayerFeedback, unless a new PlayerCmd was issued in the
// meantime.
type PlayerFeedback struct {
	CmdID int
	Err   error // If Err == nil, the command was successful.
}

type PlayerOp int

const (
	PlayerPlay PlayerOp = iota
	PlayerPause
	PlayerResume
	PlayerStop
	PlayerSeek
)
