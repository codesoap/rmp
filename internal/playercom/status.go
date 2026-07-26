package playercom

import "time"

// A Status gives information the player provides, independent of
// received commands. E.g. updates on the position in the track or song
// changes.
type Status struct {
	Type     StatusType
	Position time.Duration // Only relevant for Type == StatusPosition.
	SongID   string        // Only relevant for Type == StatusPlayingNext, StatusNowPlaying and StatusScrobble.
	Err      error         // Only relevant for Type == StatusError.
}

type StatusType int

const (
	StatusPosition StatusType = iota
	StatusPlayingNext
	StatusStopped
	StatusNowPlaying
	StatusScrobble
	StatusError
)
