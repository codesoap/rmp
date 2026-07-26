package library

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// opusReadCloser streams decoded PCM from opusdec's stdout. Closing it
// cancels the decode (if not already finished) and kills the process.
type opusReadCloser struct {
	io.Reader
	cmd    *exec.Cmd
	cancel context.CancelFunc
	stderr *bytes.Buffer
}

// Close ends the ffmpeg process and waits for it to finish.
func (r *opusReadCloser) Close() error {
	r.cancel()
	err := r.cmd.Wait()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w (stderr: %s)", err, r.stderr.String())
	}
	return nil
}

// DecodeSong decodes the file at path, producing 48kHz stereo PCM
// (WAV) on the returned reader. The caller must Close() the returned
// ReadCloser when done, even on early exit, to release the process and
// any associated resources.
//
// offset can be given to start decoding after the beginning of a song.
func DecodeSong(ctx context.Context, songID string, offset time.Duration) (io.ReadCloser, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", strconv.FormatFloat(offset.Seconds(), 'f', 3, 64),
		"-i", filepath.Join(CacheDir, songID),
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-loglevel", "fatal",
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ffmpeg: stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("ffmpeg: strt: %w", err)
	}
	return &opusReadCloser{Reader: stdout, cmd: cmd, stderr: &stderr, cancel: cancel}, nil
}
