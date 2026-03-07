package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/commands"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/handlers"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/discord/scheduler"
	appavailability "github.com/ADG08/SplatoonOrganizer/internal/application/availability"
	appguildconfig "github.com/ADG08/SplatoonOrganizer/internal/application/guildconfig"
	"github.com/ADG08/SplatoonOrganizer/internal/adapter/persistence/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	ctx, cancel := signalContext()
	defer cancel()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// Infrastructure: database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create pgx pool: %v", err)
	}
	defer pool.Close()

	// Timezone for availability
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		log.Printf("failed to load Europe/Paris location, falling back to UTC: %v", err)
		loc = time.UTC
	}

	// Adapters (secondary / driven): persistence
	availRepo := postgres.NewAvailabilityRepository(pool)
	guildConfigRepo := postgres.NewGuildConfigRepository(pool)

	// Application services (use cases)
	availSvc := appavailability.NewService(availRepo, loc)
	guildConfigSvc := appguildconfig.NewService(guildConfigRepo)

	// Adapters (primary / driving): Discord
	reg := discord.NewRegistry()

	if err := reg.RegisterCommand(commands.NewDisposCommand(availSvc)); err != nil {
		log.Fatalf("register command: %v", err)
	}

	if err := reg.RegisterHandler(handlers.NewOpenDisposHandler(availSvc)); err != nil {
		log.Fatalf("register open handler: %v", err)
	}

	if err := reg.RegisterHandler(handlers.NewSelectDisposHandler(availSvc, cfg.DiscordChannelID)); err != nil {
		log.Fatalf("register select dispos handler: %v", err)
	}

	b, err := discord.NewBot(cfg.DiscordToken, reg)
	if err != nil {
		log.Fatalf("failed to create Discord bot: %v", err)
	}
	defer b.Close()

	if err := b.Open(); err != nil {
		log.Fatalf("error opening Discord connection: %v", err)
	}

	// Scheduler for weekly posting
	weeklyPoster := scheduler.NewScheduler(availSvc, guildConfigSvc, b.Session(), cfg.DiscordGuildID, cfg.DiscordChannelID)
	if err := weeklyPoster.Start(ctx, cfg.CronSchedule); err != nil {
		log.Fatalf("start weekly scheduler: %v", err)
	}
	if cfg.RunWeeklyOnStart {
		go func() {
			if err := weeklyPoster.RunWeeklyNow(ctx); err != nil {
				log.Printf("run weekly on start: %v", err)
			}
		}()
	}

	if err := reg.RegisterCommand(commands.NewPostDisposCommand(weeklyPoster)); err != nil {
		log.Fatalf("register post-dispos command: %v", err)
	}

	if err := reg.RegisterCommand(commands.NewSetMessageChannelCommand(guildConfigSvc)); err != nil {
		log.Fatalf("register set-message-channel command: %v", err)
	}
	if err := reg.RegisterCommand(commands.NewSetRoleToPingCommand(guildConfigSvc)); err != nil {
		log.Fatalf("register set-role-to-ping command: %v", err)
	}

	if err := b.RegisterSlashCommands(cfg.DiscordClientID, cfg.DiscordGuildID); err != nil {
		log.Fatalf("failed to register slash commands: %v", err)
	}

	log.Println("SplatoonOrganizer bot is running. Press CTRL-C to exit.")

	<-ctx.Done()
	log.Println("shutting down...")
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer cancel()
		<-ch
	}()
	return ctx, cancel
}
