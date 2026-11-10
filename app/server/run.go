package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
)

// RunFromEnvironment serves the same Fiber application for the root artisan CLI
// and the optional cmd/server entry. Production-tag builds default to production
// and obtain every frontend file from the embedded bundle, independent of cwd.
func RunFromEnvironment() error {
	root := os.Getenv("APP_ROOT")
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	app, deps, err := New(root)
	if err != nil {
		return err
	}
	defer deps.Close()
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if deps.Cfg.SyncInterval > 0 {
		go deps.Syncer.Run(ctx, deps.Cfg.SyncInterval, deps.Cfg.SyncStaleAge, deps.Cfg.SyncBatchSize)
	}
	done := make(chan error, 1)
	go func() {
		log.Printf("%s on http://%s (env=%s)", deps.Cfg.SiteName, addr, deps.Env)
		done <- app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return app.ShutdownWithContext(shutdownCtx)
	}
}
