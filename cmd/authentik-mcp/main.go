package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
	appserver "github.com/zwelibam/authentik-mcp-server/internal/server"
)

func main() {
	smokeTest := flag.Bool("smoke-test", false, "Verify connectivity to Authentik and exit")
	flag.Parse()

	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	ctx := context.Background()

	if *smokeTest {
		c, err := authentik.NewClientWrapper(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "smoke-test: %v\n", err)
			os.Exit(1)
		}
		cfg, err := c.GetConfig(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "smoke-test connect failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK: connected to Authentik at %s (capabilities: %v)\n", c.BaseURL(), cfg.Capabilities)
		os.Exit(0)
	}

	if err := appserver.Run(ctx); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
