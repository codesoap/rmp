package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/codesoap/rmp/internal/config"
	"github.com/codesoap/rmp/internal/library"
	"github.com/codesoap/rmp/internal/player"
	"golang.org/x/term"
)

// TODO: Persist queue and don't ask for quit confirmation.
// TODO: Make PR at gonic to enable gzipping of JSON requests and use that.

var p player.Player
var pFlag bool

const noConfigHelp = `Create a configuration file with the following content:
    server-url = <subsonic-server-url>
    user       = <your-username>

Example:
    server-url = http://192.168.1.12:4747
    user       = admin
`

func init() {
	log.SetFlags(0)
	flag.BoolVar(&pFlag, "p", false, "change previously entered password")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-p]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Within %s, press ? to view the key map.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A configuration is expected at '%s'.\n", config.ConfigPath)
		fmt.Fprintf(os.Stderr, noConfigHelp)
		fmt.Fprintf(os.Stderr, "\nSongs are cached locally at '%s'.\n", library.CacheDir)
		fmt.Fprintf(os.Stderr, "When no song is playing, you may delete them.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
}

func main() {
	cfg := getConfigWithToken()

	log.Print("Fetching all songs initially...")
	allSongs, err := library.FetchAllSongs(cfg)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	log.Printf("Fetched %d songs.", len(allSongs))

	p, err = player.NewPlayer(cfg)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	launchUI(cfg, allSongs)
}

func getConfigWithToken() config.Config {
	cfg, err := config.LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Could not load config: %s\n\n%s", err, noConfigHelp)
	} else if err != nil {
		log.Fatal(err)
	}
	if pFlag || cfg.Token == "" {
		fmt.Printf("Password for '%s': ", cfg.User)
		pwd, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println()
		if err = cfg.CacheCredentials(cfg.User, string(pwd)); err != nil {
			log.Fatal(err)
		}
	}
	return cfg
}
