package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/Jx2f/ViaGenshin/internal/config"
	"github.com/Jx2f/ViaGenshin/internal/core"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

var c *config.Config

func init() {
	f := os.Getenv("CONFIG_FILE")
	if f == "" && len(os.Args) > 1 {
		f = os.Args[1]
	}
	if f == "" {
		p, _ := json.MarshalIndent(config.DefaultConfig, "", "  ")
		logger.Warn().Msgf("CONFIG_FILE not set, here is the default config:\n%s", p)
		os.Exit(0)
	}
	var err error
	c, err = config.LoadConfig(f)
	if err != nil {
		panic(err)
	}
	switch c.LogLevel {
	case "trace":
		logger.Logger = logger.Logger.Level(zerolog.TraceLevel)
	case "debug":
		logger.Logger = logger.Logger.Level(zerolog.DebugLevel)
	case "info":
		logger.Logger = logger.Logger.Level(zerolog.InfoLevel)
	case "silent", "disabled":
		logger.Logger = logger.Logger.Level(zerolog.Disabled)
	}
}

func main() {
	s := core.NewService(c)

	exited := make(chan error)
	go func() {
		logger.Info().Msg("Service is starting")
		exited <- s.Start()
	}()

	// Wait for a signal to quit:
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-exited:
		if err != nil {
			logger.Error().Err(err).Msg("Service exited")
		}
	case <-sig:
		logger.Info().Msg("Signal received, stopping service")
		if err := s.Stop(); err != nil {
			logger.Error().Err(err).Msg("Service stop failed")
		}
	}
}
