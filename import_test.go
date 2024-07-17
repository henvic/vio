package vio_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/henvic/pgtools/sqltest"
	"github.com/henvic/vio"
)

func TestImporter(t *testing.T) {
	t.Parallel()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("migrations"),
	})
	pool := migration.Setup(context.Background(), "")

	f, err := os.Open("testdata/example.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	stats, err := vio.NewImporter(3, slog.Default(), pool).Stream(context.Background(), f)
	if stats == nil {
		t.Error("stats should not be nil")
	}
	wantStats := &vio.ImportStats{
		Accepted:  7,
		Discarded: 4,
	}
	if diff := cmp.Diff(wantStats, stats, cmpopts.IgnoreFields(vio.ImportStats{}, "TimeElapsed")); diff != "" {
		t.Errorf("stats mismatch: %v", diff)
	}
	if err != nil {
		t.Errorf("cannot import location data: %v", err)
	}
}
