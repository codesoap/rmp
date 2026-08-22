package library

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/codesoap/rmp/internal/config"
	"github.com/codesoap/rmp/internal/song"
)

var CacheDir string

func init() {
	d, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("Could not get cache dir: %s", err)
	}
	CacheDir = filepath.Join(d, "rmp", "songs")
	if err = os.MkdirAll(CacheDir, 0700); err != nil {
		log.Fatalf("Could not create cache dir: %s", err)
	}

	// Clean up previously aborted downloads:
	matches, err := filepath.Glob(filepath.Join(CacheDir, "wip-download-*"))
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove %s: %s", path, err)
		}
	}
}

func FetchAllSongs(cfg config.Config) ([]song.Song, error) {
	u := *cfg.ServerURL
	u.Path = "/rest/search3"
	q := url.Values{}
	q.Set("u", cfg.User)
	q.Set("t", cfg.Token)
	q.Set("s", cfg.Salt)
	q.Set("c", "rmp")
	q.Set("f", "json")
	q.Set("query", `""`) // gonic does not return everything, if the quotes are missing.
	q.Set("artistCount", "0")
	q.Set("albumCount", "0")
	q.Set("songCount", "10000")
	u.RawQuery = q.Encode()
	httpClient := http.Client{Timeout: 10 * time.Second}
	rawResp, err := httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("could not fetch all songs: %s", err)
	}
	defer rawResp.Body.Close()
	resp := struct {
		SubsonicResponse struct {
			Status        string
			SearchResult3 struct {
				Song []song.Song
			}
		} `json:"subsonic-response"`
	}{}
	if err = json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("could not read all songs: %s", err)
	} else if resp.SubsonicResponse.Status != "ok" {
		return nil, fmt.Errorf("could not read all songs; status is '%s'",
			resp.SubsonicResponse.Status)
	}
	return resp.SubsonicResponse.SearchResult3.Song, nil
}

func FetchSimilarSongs(ctx context.Context, cfg config.Config, songID string) ([]song.Song, error) {
	u := *cfg.ServerURL
	u.Path = "/rest/getSimilarSongs"
	q := url.Values{}
	q.Set("u", cfg.User)
	q.Set("t", cfg.Token)
	q.Set("s", cfg.Salt)
	q.Set("c", "rmp")
	q.Set("f", "json")
	q.Set("id", songID)
	q.Set("count", "16") // TODO: Make configurable.
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not build request: %w", err)
	}
	httpClient := http.Client{Timeout: 3 * time.Second}
	rawResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch similar songs: %w", err)
	}
	defer rawResp.Body.Close()
	resp := struct {
		SubsonicResponse struct {
			Status       string
			SimilarSongs struct {
				Song []song.Song
			}
		} `json:"subsonic-response"`
	}{}
	if err = json.NewDecoder(rawResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("could not read similar songs: %s", err)
	} else if resp.SubsonicResponse.Status != "ok" {
		return nil, fmt.Errorf("could not read similar songs; status is '%s'",
			resp.SubsonicResponse.Status)
	} else if len(resp.SubsonicResponse.SimilarSongs.Song) == 0 {
		return nil, fmt.Errorf("no similar songs found")
	}
	return resp.SubsonicResponse.SimilarSongs.Song, nil
}

func Scrobble(cfg config.Config, songID string, submission bool) error {
	u := *cfg.ServerURL
	u.Path = "/rest/scrobble"
	q := url.Values{}
	q.Set("u", cfg.User)
	q.Set("t", cfg.Token)
	q.Set("s", cfg.Salt)
	q.Set("c", "rmp")
	q.Set("f", "json")
	q.Set("id", songID)
	if !submission {
		q.Set("submission", "False")
	}
	u.RawQuery = q.Encode()
	httpClient := http.Client{Timeout: 3 * time.Second}
	rawResp, err := httpClient.Get(u.String())
	if err != nil {
		return fmt.Errorf("could not scrobble: %s", err)
	}
	rawResp.Body.Close()
	return nil
}

func CacheSong(ctx context.Context, u url.URL, user, token, salt string, id string) error {
	path := filepath.Join(CacheDir, id)
	if _, err := os.Stat(path); err == nil {
		return nil // Song already exists in cache.
	}
	u.Path = "/rest/stream"
	q := url.Values{}
	q.Set("u", user)
	q.Set("t", token)
	q.Set("s", salt)
	q.Set("c", "rmp")
	q.Set("id", id)
	q.Set("format", "opus")    // TODO: Make configurable
	q.Set("maxBitRate", "160") // TODO: Make configurable
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("could not create request: %s", err)
	}
	httpClient := http.Client{} // No timeout for downloading songs.
	rawResp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not fetch song: %s", err)
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		return fmt.Errorf("got HTTP status %s", rawResp.Status)
	}
	if !filepath.IsLocal(id) {
		return fmt.Errorf("path traversal attack found")
	}
	f, err := os.CreateTemp(CacheDir, "wip-download-*")
	if err != nil {
		return err
	}
	_, err = io.Copy(f, rawResp.Body)
	f.Close()
	if err != nil {
		os.Remove(f.Name())
		return err
	}
	err = os.Rename(f.Name(), path)
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
