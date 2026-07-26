package playercom

import "github.com/codesoap/rmp/internal/song"

type QueueCmd struct {
	CmdID int
	Op    QueueOp
	Song  song.Song // Only relevant for QueueOp == QueueSetNext.
}

type QueueFeedback struct {
	CmdID int
	Err   error // If Err == nil, the queue command was successful.
}

type QueueOp int

const (
	QueueSetNext QueueOp = iota
	QueueRemoveNext
)
