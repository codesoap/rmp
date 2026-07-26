package songreader

import (
	"context"
	"io"
	"time"

	"github.com/codesoap/rmp/internal/config"
	"github.com/codesoap/rmp/internal/library"
	"github.com/codesoap/rmp/internal/song"
)

type SongReader struct {
	Song       song.Song
	cancel     context.CancelFunc
	ready      chan bool
	readCloser io.ReadCloser
	err        error
}

// NewSongReader will imediately reaturn, but its Read method will block
// until the song has been cached and decoding has begun.
func NewSongReader(cfg config.Config, song song.Song, offset time.Duration) *SongReader {
	ctx, cancel := context.WithCancel(context.Background())
	rc := &SongReader{
		Song:   song,
		cancel: cancel,
		ready:  make(chan bool),
	}
	go func() {
		defer close(rc.ready)
		err := library.CacheSong(ctx, *cfg.ServerURL, cfg.User, cfg.Token, cfg.Salt, song.ID)
		if err != nil {
			rc.err = err
			return
		}
		rc.readCloser, rc.err = library.DecodeSong(ctx, song.ID, offset)
	}()
	return rc
}

func (rc *SongReader) Read(p []byte) (int, error) {
	<-rc.ready
	if rc.err != nil {
		return 0, rc.err
	}
	return rc.readCloser.Read(p)
}

func (rc *SongReader) Close() error {
	rc.cancel()
	<-rc.ready
	return nil
}
