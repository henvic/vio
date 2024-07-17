package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/henvic/vio"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

var (
	file      = flag.String("file", "data_dump.csv", "Data dump file")
	batchSize = flag.Int("batch-size", 25000, "Batch size for the importer")
)

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
	db  *pgxpool.Pool
}

func (p *program) run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

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

	p.db, err = pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return fmt.Errorf("pgx pool connection error: %w", err)
	}

	defer p.db.Close()

	ec := make(chan error, 1)
	go func() {
		input, err := os.Open(*file)
		if err != nil {
			ec <- err
			return
		}
		defer input.Close()

		importer := vio.NewImporter(*batchSize, p.log, p.db)
		stats, err := importer.Stream(ctx, input)

		if stats != nil {
			p.log.Info("import stats", slog.Any("stats", stats))
		}

		ec <- err
		stop()
	}()

	if err = <-ec; err != nil {
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
