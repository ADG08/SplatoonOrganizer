package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // timezones of members, even on hosts without tzdata

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/commands"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/scheduler"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/weekly"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/persistence/postgres"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// key=value lines: Coolify's logs can be searched by guild, user or error.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if err := run(); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}

// run starts the bot and blocks until SIGINT or SIGTERM. Returning instead of exiting
// lets the deferred cleanups run (Discord session, database pool).
func run() error {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

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

	// Slots and publication times are in Paris time (embedded tzdata: it always loads).
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return fmt.Errorf("load Europe/Paris: %w", err)
	}

	// Adapters (secondary / driven): persistence
	availRepo := postgres.NewAvailabilityRepository(pool)
	guildConfigRepo := postgres.NewGuildConfigRepository(pool)

	// Application services (use cases)
	availSvc := appavailability.NewService(availRepo, loc)
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
		commands.NewHelpCommand(guildConfigSvc, cfg.DiscordOwnerID),
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
		handlers.NewHelpConfigHandler(guildConfigSvc, availSvc, cfg.DiscordOwnerID),
	} {
		if err := reg.RegisterHandler(h); err != nil {
			return fmt.Errorf("register handler: %w", err)
		}
	}

	b, err := discord.NewBot(cfg.DiscordToken, reg)
	if err != nil {
		return fmt.Errorf("create Discord bot: %w", err)
	}
	if err := b.Open(); err != nil {
		return err
	}
	defer b.Close()

	// Scheduler for weekly posting
	weeklyPoster := scheduler.NewScheduler(availSvc, guildConfigSvc, b.Session())
	if err := weeklyPoster.Start(ctx, cfg.RunWeeklyOnStart); err != nil {
		return fmt.Errorf("start weekly scheduler: %w", err)
	}

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

// serveHealth answers the container healthcheck until ctx is done.
func serveHealth(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, // a client sending headers slowly can't hold a connection
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("health server listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
