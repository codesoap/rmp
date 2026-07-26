package config

import (
	"bufio"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ServerURL *url.URL
	User      string

	// Salt and token are not taken from the config file, but generated
	// interactively and stored in a cache file. This avoids storing the
	// password in plain text.
	Salt  string
	Token string
}

var ConfigPath string

func init() {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Could not find config dir: %s", err)
	}
	ConfigPath = filepath.Join(cfgDir, "rmp", "rmp.conf")
}

func LoadConfig() (Config, error) {
	cfgFile, err := os.Open(ConfigPath)
	if err != nil {
		return Config{}, err
	}
	defer cfgFile.Close()
	cfg := Config{}
	scanner := bufio.NewScanner(cfgFile)
	for i := 1; scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			f := "line %d of '%s': invalid syntax: %s"
			return Config{}, fmt.Errorf(f, i, ConfigPath, line)
		}
		key = strings.TrimSpace(key)
		switch key {
		case "server-url":
			rawURL := strings.TrimSpace(value)
			cfg.ServerURL, err = url.Parse(rawURL)
			if err != nil {
				f := "invalid server URL in '%s': %s"
				return Config{}, fmt.Errorf(f, ConfigPath, err)
			}
		case "user":
			cfg.User = strings.TrimSpace(value)
		default:
			f := "line %d of '%s': unknown key: %s"
			return Config{}, fmt.Errorf(f, i, ConfigPath, key)
		}
	}

	if cfg.ServerURL == nil {
		f := "server URL not found in config at '%s'"
		return Config{}, fmt.Errorf(f, ConfigPath)
	} else if cfg.User == "" {
		f := "user not found in config at '%s'"
		return Config{}, fmt.Errorf(f, ConfigPath)
	}

	if err = cfg.loadTokenFromCache(); err != nil {
		return Config{}, fmt.Errorf("could not read token from cache: %s", err)
	}

	return cfg, nil
}

func (c *Config) loadTokenFromCache() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("could not find cache dir: %s", err)
	}
	cachePath := filepath.Join(cacheDir, "rmp", "credentials")
	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil // It's OK that there is no cache.
	}
	defer cacheFile.Close()
	scanner := bufio.NewScanner(cacheFile)

	// 1. Line is username.
	ok := scanner.Scan()
	if !ok || scanner.Text() != c.User {
		return nil // Cache is empty or old, don't use it.
	}

	// 2. Line is salt.
	ok = scanner.Scan()
	if !ok || scanner.Text() == "" {
		return fmt.Errorf("found no salt in '%s'", cachePath)
	}
	c.Salt = scanner.Text()

	// 3. Line is token.
	ok = scanner.Scan()
	if !ok || scanner.Text() == "" {
		return fmt.Errorf("found no token in '%s'", cachePath)
	}
	c.Token = scanner.Text()

	return nil
}

func (c *Config) CacheCredentials(user, password string) error {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return fmt.Errorf("could not generate salt: %s", err)
	}
	salt := fmt.Sprintf("%d", n)
	rawToken := md5.Sum([]byte(password + salt))
	token := hex.EncodeToString(rawToken[:])

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("could not find cache dir: %s", err)
	}
	appDir := filepath.Join(cacheDir, "rmp")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return fmt.Errorf("could not create cache dir: %s", err)
	}
	cachePath := filepath.Join(appDir, "credentials")
	cacheFile, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not create credentials cache: %s", err)
	}
	defer cacheFile.Close()
	if _, err = cacheFile.WriteString(user + "\n"); err != nil {
		return fmt.Errorf("could not write credentials cache: %s", err)
	} else if _, err = cacheFile.WriteString(salt + "\n"); err != nil {
		return fmt.Errorf("could not write credentials cache: %s", err)
	} else if _, err = cacheFile.WriteString(token + "\n"); err != nil {
		return fmt.Errorf("could not write credentials cache: %s", err)
	}

	c.Salt = salt
	c.Token = token
	return nil
}
