package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Development   bool
	Host          string
	Port          int
	Timezone      string
	CheckTimes    []string
	StaleAfter    time.Duration
	DataDirectory string
	UserAgent     string
	Location      *time.Location
}

func Load() (Config, error) {
	development := os.Getenv("PRICEFOLLOWER_ENV") == "development"
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
		if development {
			host = "127.0.0.1"
		}
	}
	port := 3001
	if raw := os.Getenv("PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
		}
		port = parsed
	}
	timezone := envOr("PRICEFOLLOWER_TIMEZONE", "Europe/Paris")
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Config{}, fmt.Errorf("load timezone %q: %w", timezone, err)
	}
	checkTimes := strings.Split(envOr("PRICEFOLLOWER_CHECK_TIMES", "08:00,20:00"), ",")
	for i, value := range checkTimes {
		checkTimes[i] = strings.TrimSpace(value)
		if _, err := time.Parse("15:04", checkTimes[i]); err != nil {
			return Config{}, fmt.Errorf("invalid collection time %q; expected HH:MM", checkTimes[i])
		}
	}
	staleHours := 36
	if raw := os.Getenv("PRICEFOLLOWER_STALE_AFTER_HOURS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return Config{}, fmt.Errorf("PRICEFOLLOWER_STALE_AFTER_HOURS must be a positive integer")
		}
		staleHours = parsed
	}
	dataDirectory := os.Getenv("PRICEFOLLOWER_DATA_DIR")
	if dataDirectory == "" {
		if development {
			dataDirectory = ".data"
		} else {
			dataDirectory = "/var/lib/pricefollower"
		}
	}
	userAgent := envOr("AMAZON_USER_AGENT", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/154.0.0.0 Safari/537.36")
	return Config{
		Development:   development,
		Host:          host,
		Port:          port,
		Timezone:      timezone,
		CheckTimes:    checkTimes,
		StaleAfter:    time.Duration(staleHours) * time.Hour,
		DataDirectory: filepath.Clean(dataDirectory),
		UserAgent:     userAgent,
		Location:      location,
	}, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
