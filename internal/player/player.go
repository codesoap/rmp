// Package playes provides a music player, which also downloads
// and caches songs from a Subsonic server.
package player

import (
	"time"

	"github.com/codesoap/rmp/internal/config"
	"github.com/codesoap/rmp/internal/player/songreader"
	"github.com/codesoap/rmp/internal/player/streamplayer"
	"github.com/codesoap/rmp/internal/playercom"
	"github.com/codesoap/rmp/internal/song"
)

// FIXME: Avoid restarting downloads, e.g. when play spamming.

// Player implements a gapless music player, which also downloads and
// caches songs.
//
// Listen to its events for status updates (playing, paused, play
// duration, errors, ...).
type Player struct {
	cfg            config.Config
	streamPlayer   streamplayer.StreamPlayer
	playerCmds     chan playercom.PlayerCmd
	queueCmds      chan playercom.QueueCmd
	playerFeedback chan playercom.PlayerFeedback
	queueFeedback  chan playercom.QueueFeedback
	status         chan playercom.Status
}

func NewPlayer(cfg config.Config) (Player, error) {
	status := make(chan playercom.Status, 32)
	sp, err := streamplayer.NewStreamPlayer(status)
	if err != nil {
		return Player{}, err
	}
	player := Player{
		cfg:        cfg,
		playerCmds: make(chan playercom.PlayerCmd),

		// Queuing often happens directly after playing a song, so
		// give the channel a minimal buffer:
		queueCmds: make(chan playercom.QueueCmd, 1),

		playerFeedback: make(chan playercom.PlayerFeedback, 32),
		queueFeedback:  make(chan playercom.QueueFeedback, 32),
		status:         status,
		streamPlayer:   sp,
	}
	go player.runEventLoop()
	return player, nil
}

func (p Player) PlayerCmds() chan<- playercom.PlayerCmd          { return p.playerCmds }
func (p Player) QueueCmds() chan<- playercom.QueueCmd            { return p.queueCmds }
func (p Player) PlayerFeedback() <-chan playercom.PlayerFeedback { return p.playerFeedback }
func (p Player) QueueFeedback() <-chan playercom.QueueFeedback   { return p.queueFeedback }
func (p Player) Status() <-chan playercom.Status                 { return p.status }

func (p *Player) runEventLoop() {
	var seekLimiter *time.Timer
	for {
		select {
		case cmd := <-p.playerCmds:
			switch cmd.Op {
			case playercom.PlayerPlay:
				err := p.play(cmd.Song)
				p.playerFeedback <- playercom.PlayerFeedback{CmdID: cmd.CmdID, Err: err}
			case playercom.PlayerPause:
				p.streamPlayer.Pause()
				p.playerFeedback <- playercom.PlayerFeedback{CmdID: cmd.CmdID}
			case playercom.PlayerResume:
				p.streamPlayer.Resume()
				p.playerFeedback <- playercom.PlayerFeedback{CmdID: cmd.CmdID}
			case playercom.PlayerStop:
				p.streamPlayer.Stop()
				p.playerFeedback <- playercom.PlayerFeedback{CmdID: cmd.CmdID}
			case playercom.PlayerSeek:
				if seekLimiter != nil {
					seekLimiter.Stop()
				}
				seekLimiter = time.AfterFunc(300*time.Millisecond, func() {
					current := songreader.NewSongReader(p.cfg, cmd.Song, cmd.SeekTo)
					p.streamPlayer.Seek(current, cmd.SeekTo)
					p.playerFeedback <- playercom.PlayerFeedback{CmdID: cmd.CmdID}
				})
			}
		case cmd := <-p.queueCmds:
			var err error
			switch cmd.Op {
			case playercom.QueueSetNext:
				err = p.setNext(cmd.Song)
			case playercom.QueueRemoveNext:
				p.streamPlayer.RemoveNext()
			}
			p.queueFeedback <- playercom.QueueFeedback{CmdID: cmd.CmdID, Err: err}
		}
	}
}

// play plays the given song. It is downloaded to the cache first, if
// necessary; the Status method will report Loading == true in this
// case.
//
// Play returns quickly, doing the download and playback concurrently.
// Playback can be canceled by calling Stop or calling Play again.
func (p *Player) play(song song.Song) error {
	current := songreader.NewSongReader(p.cfg, song, 0)
	p.streamPlayer.Play(current)
	return nil
}

func (p *Player) setNext(song song.Song) error {
	next := songreader.NewSongReader(p.cfg, song, 0)
	return p.streamPlayer.SetNext(next)
}
