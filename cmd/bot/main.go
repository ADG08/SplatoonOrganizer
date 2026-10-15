package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/commands"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/scheduler"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/persistence/postgres"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// "bot healthcheck" is the container's healthcheck: the image has no shell nor wget.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(checkHealth())
	}

	// key=value lines: Coolify's logs can be searched by guild, user or error.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if err := run(); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}

// run starts the bot and blocks until SIGINT or SIGTERM. Returning instead of exiting
// lets the deferred cleanups run (Discord session, database pool). The configuration
// comes from the environment (Docker Compose reads .env, Coolify its own variables).
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Infrastructure: database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create pgx pool: %w", err)
	}
	defer pool.Close()

	// Adapters (secondary / driven): persistence
	availRepo := postgres.NewAvailabilityRepository(pool)
	guildConfigRepo := postgres.NewGuildConfigRepository(pool)

	// Application services (use cases)
	availSvc := appavailability.NewService(availRepo)
	guildConfigSvc := appguildconfig.NewService(guildConfigRepo)

	// Single-guild versions took the weekly channel from the environment: keep it as that
	// guild's configured channel, the only source of channels now.
	if cfg.DiscordGuildID != "" && cfg.DiscordChannelID != "" {
		gc, err := guildConfigSvc.Get(ctx, cfg.DiscordGuildID)
		if err != nil {
			return fmt.Errorf("get guild config: %w", err)
		}
		if gc.ChannelID == "" {
			if err := guildConfigSvc.SetChannel(ctx, cfg.DiscordGuildID, cfg.DiscordChannelID); err != nil {
				return fmt.Errorf("set guild channel from env: %w", err)
			}
		}
	}

	// Adapters (primary / driving): Discord
	reg := discord.NewRegistry()
	refresher := weekly.NewRefresher(availSvc, guildConfigSvc)
	for _, c := range []discord.Command{
		commands.NewHelpCommand(guildConfigSvc, cfg.DiscordOwnerID, cfg.SupportURL),
		commands.NewAnnonceCommand(guildConfigSvc, cfg.DiscordOwnerID),
	} {
		if err := reg.RegisterCommand(c); err != nil {
			return fmt.Errorf("register command: %w", err)
		}
	}
	for _, h := range []discord.InteractionHandler{
		handlers.NewOpenDisposHandler(availSvc, guildConfigSvc),
		handlers.NewViewDisposHandler(availSvc, guildConfigSvc),
		handlers.NewTimezoneSelectHandler(availSvc, guildConfigSvc),
		handlers.NewTimezoneEditHandler(guildConfigSvc),
		handlers.NewMyTimeHandler(availSvc, guildConfigSvc),
		handlers.NewWhoDisposHandler(availSvc, guildConfigSvc),
		handlers.NewSelectDisposHandler(availSvc, guildConfigSvc, refresher),
		handlers.NewUnavailableDisposHandler(availSvc, guildConfigSvc, refresher),
		handlers.NewHelpConfigHandler(guildConfigSvc, availSvc, refresher, cfg.DiscordOwnerID, cfg.SupportURL),
	} {
		if err := reg.RegisterHandler(h); err != nil {
			return fmt.Errorf("register handler: %w", err)
		}
	}

	b, err := discord.NewBot(cfg.DiscordToken, reg)
	if err != nil {
		return fmt.Errorf("create Discord bot: %w", err)
	}
	membership := handlers.NewGuildMembership(guildConfigSvc)
	b.Session().AddHandler(membership.OnGuildCreate)
	b.Session().AddHandler(membership.OnGuildDelete)
	if err := b.Open(); err != nil {
		return err
	}
	defer b.Close()

	// Scheduler for weekly posting
	scheduler.NewScheduler(availSvc, guildConfigSvc, b.Session()).Start(ctx, cfg.RunWeeklyOnStart)

	if err := b.RegisterSlashCommands(cfg.DiscordClientID, cfg.DiscordGuildID); err != nil {
		return fmt.Errorf("register slash commands: %w", err)
	}

	go func() {
		if err := serveHealth(ctx, cfg.HealthPort); err != nil {
			slog.Error("health server stopped", "err", err)
		}
	}()

	slog.Info("SplatoonOrganizer bot is running. Press CTRL-C to exit.")
	<-ctx.Done()
	slog.Info("shutting down...")
	return nil
}
