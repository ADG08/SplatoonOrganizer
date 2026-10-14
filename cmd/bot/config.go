package main

import "os"

// Config holds application configuration from environment.
type Config struct {
	DiscordToken     string
	DiscordClientID  string
	DiscordGuildID   string // Optional: the guild of single-guild versions, migrated on startup (see main).
	DiscordChannelID string // Optional: that guild's channel, used when /help never set one.
	DiscordOwnerID   string // Optional: Discord user ID of the bot creator (can use admin commands without being server admin).
	DatabaseURL      string
	RunWeeklyOnStart bool
	HealthPort       string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8080"
	}

	cfg := &Config{
		DiscordToken:     os.Getenv("DISCORD_TOKEN"),
		DiscordClientID:  os.Getenv("DISCORD_CLIENT_ID"),
		DiscordGuildID:   os.Getenv("DISCORD_GUILD_ID"),
		DiscordChannelID: os.Getenv("DISCORD_CHANNEL_ID"),
		DiscordOwnerID:   os.Getenv("DISCORD_OWNER_ID"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RunWeeklyOnStart: os.Getenv("RUN_WEEKLY_ON_START") == "1",
		HealthPort:       healthPort,
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
