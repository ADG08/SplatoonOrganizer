package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/commands"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/logwebhook"
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
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, nil)
	// LOG_WEBHOOK_URL, optional: the logs are also posted in a channel of the owner's server.
	var logs *logwebhook.Handler
	if webhookURL := os.Getenv("LOG_WEBHOOK_URL"); webhookURL != "" {
		var err error
		if logs, err = logwebhook.New(webhookURL, handler); err != nil {
			slog.New(handler).Warn("LOG_WEBHOOK_URL ignored", "err", err)
		} else {
			handler = logs
		}
	}
	slog.SetDefault(slog.New(handler))

	err := run(logs)
	if err != nil {
		slog.Error("bot stopped", "err", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	logs.Close(ctx)
	cancel()
	if err != nil {
		os.Exit(1)
	}
}

// run starts the bot and blocks until SIGINT or SIGTERM. Returning instead of exiting
// lets the deferred cleanups run (Discord session, database pool). The configuration
// comes from the environment (Docker Compose reads .env, Coolify its own variables).
// logs, when set, gets the guild names once the Discord session exists.
func run(logs *logwebhook.Handler) error {
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
	logs.SetGuildNames(func(guildID string) string {
		if g, err := b.Session().State.Guild(guildID); err == nil {
			return g.Name
		}
		return ""
	})
	membership := handlers.NewGuildMembership(guildConfigSvc)
	b.Session().AddHandler(membership.OnGuildCreate)
	b.Session().AddHandler(membership.OnGuildDelete)
	if err := b.Open(); err != nil {
		return err
	}
	defer b.Close()

	// Scheduler for weekly posting
	scheduler.NewScheduler(availSvc, guildConfigSvc, b.Session()).Start(ctx, cfg.RunWeeklyOnStart)

	if err := b.RegisterSlashCommands(cfg.DiscordClientID); err != nil {
		return fmt.Errorf("register slash commands: %w", err)
	}

	go func() {
		if err := serveHealth(ctx, cfg.HealthPort); err != nil {
			slog.Error("health server stopped", "err", err)
		}
	}()

	slog.Info("bot started")
	<-ctx.Done()
	slog.Info("bot shutting down")
	return nil
}
