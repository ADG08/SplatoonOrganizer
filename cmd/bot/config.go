package main

import "os"

// defaultHealthPort is the port of /health when HEALTH_PORT is not set.
const defaultHealthPort = "8080"

// Config holds application configuration from environment.
type Config struct {
	DiscordToken     string
	DiscordClientID  string
	DiscordOwnerID   string // Optional: Discord user ID of the bot creator (can use admin commands without being server admin).
	DatabaseURL      string
	RunWeeklyOnStart bool
	HealthPort       string
	SupportURL       string // Optional: donation page, linked from the /help panel.
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = defaultHealthPort
	}

	cfg := &Config{
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		DiscordClientID:  os.Getenv("DISCORD_CLIENT_ID"),
		DiscordOwnerID:   os.Getenv("DISCORD_OWNER_ID"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RunWeeklyOnStart: os.Getenv("RUN_WEEKLY_ON_START") == "1",
		HealthPort:       healthPort,
		SupportURL:       os.Getenv("SUPPORT_URL"),
	}

	if cfg.DiscordToken == "" {
		return nil, ErrEnv("DISCORD_TOKEN is required")
	}
	if cfg.DiscordClientID == "" {
		return nil, ErrEnv("DISCORD_CLIENT_ID is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, ErrEnv("DATABASE_URL is required")
	}
	return cfg, nil
}

// EnvError represents an environment variable configuration error.
type EnvError struct {
	Msg string
}

func (e EnvError) Error() string {
	return e.Msg
}

// ErrEnv returns an error for missing or invalid environment variables.
func ErrEnv(msg string) error {
	return EnvError{Msg: msg}
}
