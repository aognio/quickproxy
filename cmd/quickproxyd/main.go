package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/gnrfan/quickproxy/internal/config"
	"github.com/gnrfan/quickproxy/internal/doctor"
	"github.com/gnrfan/quickproxy/internal/logging"
	"github.com/gnrfan/quickproxy/internal/server"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := flag.String("config", "/etc/quickproxy/server-config.toml", "Path to configuration file")
	doctorMode := flag.Bool("doctor", false, "Run diagnostic checks")
	doctorFormat := flag.String("doctor-format", "text", "Output format for doctor (text or json)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("quickproxyd %s\n", version)
		if commit != "" {
			fmt.Printf("Commit: %s\n", commit)
		}
		if date != "" {
			fmt.Printf("Built: %s\n", date)
		}
		return
	}

	if *doctorMode {
		d := doctor.New(*configPath)
		output := d.Run(*doctorFormat)
		fmt.Print(output)
		return
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	log.Info("quickproxyd starting",
		"version", version,
		"config", *configPath,
	)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("failed to load configuration", "error", err.Error())
		os.Exit(1)
	}

	log.Info("configuration loaded",
		"listen", cfg.ListenAddress,
		"domains", cfg.Domains,
		"target_host", cfg.TargetHost,
	)

	appLogger, err := logging.New(cfg.LogDirectory)
	if err != nil {
		log.Error("failed to initialize logging", "error", err.Error())
		os.Exit(1)
	}

	srv := server.New(cfg, appLogger, log)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := srv.Start(); err != nil {
			log.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	<-sigChan
	log.Info("received shutdown signal")

	if err := srv.Shutdown(); err != nil {
		log.Error("shutdown error", "error", err.Error())
		os.Exit(1)
	}
}

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				commit = setting.Value
			case "vcs.time":
				date = setting.Value
			}
		}
	}
}
