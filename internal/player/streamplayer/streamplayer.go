// Package streamplayer provides a gapless music player that works
// with io.ReadCloser.
package streamplayer

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/codesoap/rmp/internal/player/songreader"
	"github.com/codesoap/rmp/internal/playercom"
	"github.com/ebitengine/oto/v3"
)

type StreamPlayer struct {
	mu            *sync.Mutex // FIXME: Might be uneccesarry, if event loop serializes.
	muNext        *sync.Mutex
	current       *songreader.SongReader
	atFrame       int // The read frames in current (48000 fps).
	reportedFrame int // Last reported position with Position event.
	readFrames    int // Totally read frames in current.
	next          *songreader.SongReader
	otoCtx        *oto.Context
	player        *oto.Player
	Status        chan<- playercom.Status
}

func NewStreamPlayer(status chan<- playercom.Status) (StreamPlayer, error) {
	if err := exec.Command("pulseaudio", "--start").Run(); err != nil {
		return StreamPlayer{}, fmt.Errorf("could not start pulseaudio: %s", err)
	}
	otoCtx, rdy, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   48000, // My decoder always gives 48 kHz.
		ChannelCount: 2,     // My decoder always gives stereo.
		Format:       oto.FormatSignedInt16LE,
		// Choosing a small buffer, because it is still played after pause:
		BufferSize: time.Millisecond * 50,
	})
	if err != nil {
		return StreamPlayer{}, err
	}
	<-rdy
	return StreamPlayer{
		mu:     &sync.Mutex{},
		muNext: &sync.Mutex{},
		otoCtx: otoCtx,
		Status: status,
	}, nil
}

func (fp *StreamPlayer) Read(p []byte) (n int, err error) {
	// fp.mu is not locked in here at all, since there is no place where
	// fp.player is manipulated without previously stopping reads (through
	// fp.player.PauseAndStopReading).

	if fp.current == nil {
		return 0, io.EOF
	}
	n, err = fp.current.Read(p)

	if fp.readFrames == 0 && n > 0 {
		fp.Status <- playercom.Status{
			Type:   playercom.StatusNowPlaying,
			SongID: fp.current.Song.ID,
		}
	} else if n > 0 && (fp.readFrames+n)/48_000/4 >= 30 {
		scrobbleThreshold := min(4*time.Minute,
			time.Duration(fp.current.Song.Duration)*time.Second/2)
		playtimeBefore := time.Millisecond * time.Duration(fp.readFrames/48/4)
		playtimeAfter := time.Millisecond * time.Duration((fp.readFrames+n)/48/4)
		if playtimeBefore < scrobbleThreshold && playtimeAfter >= scrobbleThreshold {
			// Song is at least 30s song and played for at least 4min or half its
			// length.
			fp.Status <- playercom.Status{
				Type:   playercom.StatusScrobble,
				SongID: fp.current.Song.ID,
			}
		}
	}
	fp.atFrame += n
	fp.readFrames += n
	if fp.reportedFrame == 0 || fp.atFrame-fp.reportedFrame > 48_000*4 {
		fp.Status <- playercom.Status{
			Type:     playercom.StatusPosition,
			Position: time.Second * time.Duration(fp.atFrame/48_000/4),
		}
		fp.reportedFrame = fp.atFrame
	}
	if err != nil {
		fp.current.Close()
		fp.current = nil
		if err != io.EOF {
			fp.Status <- playercom.Status{Type: playercom.StatusError, Err: err}
		}
		fp.muNext.Lock()
		defer fp.muNext.Unlock()
		if fp.next == nil {
			fp.player.Pause()
			fp.player = nil
			fp.Status <- playercom.Status{Type: playercom.StatusStopped}
			return
		}
		fp.current = fp.next
		fp.next = nil
		fp.Status <- playercom.Status{
			Type:   playercom.StatusPlayingNext,
			SongID: fp.current.Song.ID,
		}
		n, err = fp.current.Read(p)
		fp.atFrame = n
		fp.readFrames = n
		fp.reportedFrame = 0
		if n > 0 {
			fp.Status <- playercom.Status{
				Type:   playercom.StatusNowPlaying,
				SongID: fp.current.Song.ID,
			}
		}
		if err != nil {
			fp.current.Close()
			fp.current = nil
			if err != io.EOF {
				fp.Status <- playercom.Status{Type: playercom.StatusError, Err: err}
			}
			fp.player.Pause()
			fp.player = nil
			fp.Status <- playercom.Status{Type: playercom.StatusStopped}
		} else {
			fp.Status <- playercom.Status{Type: playercom.StatusPosition, Position: 0}
		}
	}
	return
}

func (fp *StreamPlayer) Play(sr *songreader.SongReader) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if fp.player != nil {
		fp.player.PauseAndStopReading()
		fp.player = nil
	}
	if fp.current != nil {
		// Drop error, because it has no meaning to a user.
		_ = fp.current.Close()
	}
	if fp.next != nil {
		// Drop error, because it has no meaning to a user.
		_ = fp.next.Close()
		fp.next = nil
	}
	fp.current = sr
	fp.player = fp.otoCtx.NewPlayer(fp)
	fp.player.Play()
	fp.atFrame = 0
	fp.readFrames = 0
	fp.reportedFrame = 0
}

func (fp *StreamPlayer) Pause() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if fp.player != nil {
		fp.player.Pause()
	}
}

func (fp *StreamPlayer) Reset() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if fp.player != nil {
		fp.player.PauseAndStopReading()
	}
}

func (fp *StreamPlayer) Resume() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if fp.player != nil {
		fp.player.Play()
	}
}

func (fp *StreamPlayer) Stop() {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	if fp.player != nil {
		fp.player.PauseAndStopReading()
		fp.player = nil
	}
	if fp.current != nil {
		fp.Status <- playercom.Status{Type: playercom.StatusStopped}
		// Drop error, because it has no meaning to a user.
		_ = fp.current.Close()
		fp.current = nil
	}
	if fp.next != nil {
		// Drop error, because it has no meaning to a user.
		_ = fp.next.Close()
		fp.next = nil
	}
}

func (fp *StreamPlayer) Seek(sr *songreader.SongReader, offset time.Duration) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	var wasPlaying bool
	if fp.player != nil {
		wasPlaying = fp.player.IsPlaying()
		fp.player.PauseAndStopReading()
		fp.player = nil
	}
	if fp.current != nil {
		_ = fp.current.Close()
	}
	fp.current = sr
	fp.player = fp.otoCtx.NewPlayer(fp)
	if wasPlaying {
		fp.player.Play()
	}
	fp.atFrame = int(4 * 48 * int64(offset) / int64(time.Millisecond))
	// fp.readFrames is not changed here, as it cares about total playtime.
	fp.reportedFrame = fp.atFrame
	fp.Status <- playercom.Status{Type: playercom.StatusPosition, Position: offset}
}

func (fp *StreamPlayer) SetNext(sr *songreader.SongReader) error {
	fp.mu.Lock()
	fp.muNext.Lock()
	defer fp.mu.Unlock()
	defer fp.muNext.Unlock()
	if fp.current == nil {
		return fmt.Errorf("cannot set next song, while current is not set")
	}
	if fp.next != nil {
		// Drop error, because it has no meaning to a user.
		_ = fp.next.Close()
	}
	fp.next = sr
	return nil
}

func (fp *StreamPlayer) RemoveNext() {
	fp.mu.Lock()
	fp.muNext.Lock()
	defer fp.mu.Unlock()
	defer fp.muNext.Unlock()
	if fp.next != nil {
		// Drop error, because it has no meaning to a user.
		_ = fp.next.Close()
		fp.next = nil
	}
}
