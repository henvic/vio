package vio

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportStats of the geolocation ingestion.
type ImportStats struct {
	TimeElapsed time.Duration
	Accepted    int
	Discarded   int
}

// NewImporter creates a new CSV reader.
func NewImporter(batchSize int, log *slog.Logger, db *pgxpool.Pool) *Importer {
	return &Importer{
		batchSize: batchSize,
		log:       log,
		db:        db,
	}
}

// Importer for the CSV stream.
type Importer struct {
	// batchSize is the number of records to be inserted in a single batch.
	batchSize int

	// log for the importer.
	log *slog.Logger

	// pool for accessing Postgres database.PGX
	db *pgxpool.Pool
}

// Stream imports data from CSV input and stream it to database.
//
// Assume the typical CSV format is
// ip_address,country_code,country,city,latitude,longitude,mystery_value
func (i *Importer) Stream(ctx context.Context, r io.Reader) (*ImportStats, error) {
	var (
		stats ImportStats
		begin = time.Now()
	)

	defer func() {
		stats.TimeElapsed = time.Since(begin)
	}()

	// Use batches to reduce round-trips.
	var (
		batch       pgx.Batch
		batchNumber int
		total       int
	)

	stream := csv.NewReader(r)
	stream.ReuseRecord = true

	for {
		if ctx.Err() != nil {
			return &stats, ctx.Err()
		}

		record, err := stream.Read()
		if err == io.EOF {
			break
		}
		if err != nil && err != csv.ErrFieldCount {
			stats.Discarded++
			continue
		}

		var loc Geolocation
		if err := i.loadRecord(record, &loc); err != nil {
			stats.Discarded++
			continue
		}

		// NOTE(henvic): PostgreSQL treats IPv4 and IPv6 versions of the same IP differently.
		// pgx is normalizing it to IPv6.
		batch.Queue(importQuery,
			loc.IPAddress,
			loc.CountryCode,
			loc.Country,
			loc.City,
			loc.Latitude,
			loc.Longitude)

		if batch.Len() == i.batchSize {
			batchNumber++
			total += batch.Len()
			results := i.db.SendBatch(ctx, &batch)
			if err := results.Close(); err != nil {
				return &stats, fmt.Errorf("batch %d error: %w", batchNumber, err)
			}
			i.log.Info("Batch processed",
				slog.Int("batch", batchNumber),
				slog.Any("total", total),
			)

			// Recreate a new batch.
			batch = pgx.Batch{}
		}

		stats.Accepted++
	}

	if batch.Len() > 0 {
		batchNumber++
		total += batch.Len()
		results := i.db.SendBatch(ctx, &batch)

		if err := results.Close(); err != nil {
			return &stats, fmt.Errorf("batch %d error: %w", batchNumber, err)
		}
		i.log.Info("Batch processed",
			slog.Int("batch", batchNumber),
			slog.Any("total", total),
		)
	}

	return &stats, nil
}

// loadRecord into the Geolocation struct.
func (i *Importer) loadRecord(record []string, loc *Geolocation) error {
	// Try to find the IP field.
	for pos, v := range record {
		if ip := net.ParseIP(v); ip != nil {
			loc.IPAddress = ip
			record = append(record[:pos], record[pos+1:]...)
		}
	}
	if len(loc.IPAddress) == 0 {
		return errors.New("no valid IP address found")
	}

	// Try to find the latitude and longitude fields.
	if len(record) >= 2 {
		for pos := 0; pos < len(record)-1; pos++ {
			lat, err := strconv.ParseFloat(record[pos], 64)
			if err != nil || record[pos] == "0" || lat < -90 || lat > 90 {
				continue
			}
			lon, err := strconv.ParseFloat(record[pos+1], 64)
			if err != nil || record[pos+1] == "0" || lon < -180 || lon > 180 {
				continue
			}
			// Maintain exact precision for coordinates.
			loc.Latitude = json.Number(record[pos])
			loc.Longitude = json.Number(record[pos+1])
			record = append(record[:pos], record[pos+2:]...)
			break
		}
	}

	// Try to find the country code field.
	for pos, v := range record {
		if isCountryCode(record[pos]) {
			loc.CountryCode = v
			record = append(record[:pos], record[pos+1:]...)
			break
		}
	}

	// Try to find country and city.
	if len(record) >= 1 {
		loc.Country = record[0]
	}
	if len(record) >= 2 {
		loc.City = record[1]
	}

	// If no useful values are found, assume that the data is corrupted.
	if loc.City == "" && loc.Country == "" && loc.CountryCode == "" &&
		loc.Latitude == "" && loc.Longitude == "" {
		return errors.New("no useful data found")
	}
	// Otherwise, accept the record.
	return nil
}

// isCountryCode naively checks if the given string is an uppercase 2-letter ISO 3166-1 country code.
func isCountryCode(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, c := range code {
		if !unicode.IsUpper(c) {
			return false
		}
	}
	return true
}
