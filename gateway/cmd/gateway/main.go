// Command gateway wires config, database, auth, the Signal service and the
// HTTP server together.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"github.com/DonovanDiamond/signalmgr/gateway/src/api"
	"github.com/DonovanDiamond/signalmgr/gateway/src/auth"
	"github.com/DonovanDiamond/signalmgr/gateway/src/config"
	"github.com/DonovanDiamond/signalmgr/gateway/src/db"
	sig "github.com/DonovanDiamond/signalmgr/gateway/src/signal"
	"github.com/DonovanDiamond/signalmgr/gateway/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file; a missing file means all defaults")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	x, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("opening database")
	}
	defer x.Close()

	authSvc, err := auth.New(cfg, x, log)
	if err != nil {
		log.Fatal().Err(err).Msg("starting auth")
	}
	signalSvc := sig.New(x, cfg.SignalAPIURL, log)

	// On error FS still returns a page explaining that the UI is not built.
	webFS, err := web.FS()
	if err != nil {
		log.Warn().Err(err).Msg("serving the placeholder page instead of the UI")
	}
	app := api.New(authSvc, signalSvc, cfg.InstallationName, cfg.TrustedProxies, webFS, log)

	go func() {
		if err := app.Listen(cfg.Listen, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			log.Fatal().Err(err).Msg("server stopped unexpectedly")
		}
	}()
	log.Info().Str("listen", cfg.Listen).Str("signal_api_url", cfg.SignalAPIURL).Msg("gateway listening")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info().Msg("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
}

// loadConfig treats a missing file as "use the defaults".
func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return config.Defaults().Resolve()
	}
	return config.Load(path)
}
