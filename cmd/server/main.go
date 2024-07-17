package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/henvic/vio"
	"github.com/henvic/vio/internal/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

var httpAddr = flag.String("http", "localhost:8080", "HTTP service address to listen for incoming requests on")

func main() {
	flag.Parse()
	p := program{
		log: slog.Default(),
	}

	if err := p.run(); err != nil {
		p.log.Error("application terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

type program struct {
	log *slog.Logger
}

func (p *program) run() error {
	// Using environment variables instead of a connection string.
	// Reference for PostgreSQL environment variables:
	// https://www.postgresql.org/docs/current/libpq-envars.html
	conf, err := pgxpool.ParseConfig("")
	if err != nil {
		return err
	}

	conf.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   pgxLogger{log: p.log},
		LogLevel: tracelog.LogLevelError,
	}

	db, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		return fmt.Errorf("pgx pool connection error: %w", err)
	}

	defer db.Close()

	s := api.NewServer(*httpAddr, vio.NewService(vio.NewPostgres(db, p.log)), p.log)
	ec := make(chan error, 1)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		ec <- s.Run(context.Background())
	}()

	// Waits for an internal error that shutdowns the server.
	// Otherwise, wait for a SIGINT or SIGTERM and tries to shutdown the server gracefully.
	// After a shutdown signal, HTTP requests taking longer than the specified grace period are forcibly closed.
	select {
	case err = <-ec:
	case <-ctx.Done():
		fmt.Println()
		haltCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.Shutdown(haltCtx)
		stop()
		err = <-ec
	}
	if err != nil {
		return err
	}
	return nil
}

// pgxLogger prints pgx logs to the standard logger.
// os.Stderr by default.
type pgxLogger struct {
	log *slog.Logger
}

func (l pgxLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	attrs := make([]slog.Attr, 0, len(data)+1)
	attrs = append(attrs, slog.String("pgx_level", level.String()))
	for k, v := range data {
		attrs = append(attrs, slog.Any(k, v))
	}
	l.log.LogAttrs(ctx, slogLevel(level), msg, attrs...)
}

// slogLevel translates pgx log level to slog log level.
func slogLevel(level tracelog.LogLevel) slog.Level {
	switch level {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		return slog.LevelDebug
	case tracelog.LogLevelInfo:
		return slog.LevelInfo
	case tracelog.LogLevelWarn:
		return slog.LevelWarn
	default:
		// If tracelog.LogLevelError, tracelog.LogLevelNone, or any other unknown level, use slog.LevelError.
		return slog.LevelError
	}
}
