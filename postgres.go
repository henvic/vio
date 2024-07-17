package vio

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/henvic/pgtools"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgres creates a new Postgres database.
func NewPostgres(pool *pgxpool.Pool, log *slog.Logger) *Postgres {
	return &Postgres{pool: pool, log: log}
}

// Postgres database implementation.
type Postgres struct {
	// pool for accessing Postgres database.PGX
	pool *pgxpool.Pool

	// log is a log for the operations.
	log *slog.Logger
}

// lookupLocationQuery used to get a geolocation from the database.
// Or rather:
// var lookupLocationQuery = `SELECT ip_address,country_code,country,city,latitude,longitude,updated_at FROM geolocation WHERE ip_address = $1 LIMIT 1;`
var lookupLocationQuery = `SELECT ` + pgtools.Wildcard(Geolocation{}) + ` FROM geolocation WHERE ip_address = $1 LIMIT 1;`

// LookupLocation returns a location.
func (pg Postgres) LookupLocation(ctx context.Context, ip net.IP) (*Geolocation, error) {
	rows, err := pg.pool.Query(ctx, lookupLocationQuery, ip)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	var loc Geolocation
	if err == nil {
		loc, err = pgx.CollectOneRow(rows, pgx.RowToStructByPos[Geolocation])
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		pg.log.Error("cannot get location from database",
			slog.Any("ip", ip),
			slog.Any("error", err),
		)
		return nil, errors.New("cannot get location from database")
	}
	return &loc, nil
}

// importQuery used to insert data into the database.
const importQuery = `INSERT INTO geolocation (
ip_address, country_code, country, city, latitude, longitude
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (ip_address) DO UPDATE SET
country_code = EXCLUDED.country_code,
country = EXCLUDED.country,
city = EXCLUDED.city,
latitude = EXCLUDED.latitude,
longitude = EXCLUDED.longitude,
updated_at = now();`
