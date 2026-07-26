package song

import (
	"fmt"
	"strings"
)

type Song struct {
	ID       string
	Title    string
	Album    string
	Artist   string
	Track    int
	Duration int // Duration in seconds.
}

func (s Song) String() string {
	var sb strings.Builder
	if s.Artist != "" {
		sb.WriteString(s.Artist)
		if s.Album != "" && s.Album != "Unknown Album" {
			sb.WriteString(" - " + s.Album)
			if s.Track > 0 {
				fmt.Fprintf(&sb, " [#%02d]", s.Track)
			}
		}
	}
	if s.Title != "" {
		if sb.Len() > 0 {
			sb.WriteString(" - ")
		}
		sb.WriteString(s.Title)
	}
	return sb.String()
}
