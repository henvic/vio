package api

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/henvic/pgtools/sqltest"
	"github.com/henvic/vio"
)

var force = flag.Bool("force", false, "Force cleaning the database before starting")

func TestLookup(t *testing.T) {
	migration := sqltest.New(t, sqltest.Options{
		Force:                   *force,
		Files:                   os.DirFS("../../migrations"),
		TemporaryDatabasePrefix: "test_vio_api",
	})
	pool := migration.Setup(context.Background(), "")

	file, err := os.Open("../../testdata/example.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	stats, err := vio.NewImporter(3, slog.Default(), pool).Stream(context.Background(), file)
	if stats == nil {
		t.Error("stats should not be nil")
	}
	if err != nil {
		t.Errorf("cannot import location data: %v", err)
	}

	s := NewServer("", vio.NewService(vio.NewPostgres(pool, slog.Default())), slog.Default())
	hs := httptest.NewServer(http.HandlerFunc(s.lookupHandler))

	type args struct {
		ip string
	}
	tests := []struct {
		name string
		args args
		loc  *vio.Geolocation
		err  *APIError
	}{
		{
			name: "empty_ip",
			args: args{
				ip: "",
			},
			err: &APIError{
				HTTPCode: http.StatusBadRequest,
				Message:  "missing mandatory IP address query param",
			},
		},
		{
			name: "bad_ip",
			args: args{
				ip: "x",
			},
			err: &APIError{
				HTTPCode: http.StatusBadRequest,
				Message:  "invalid IP address format",
			},
		},
		{
			name: "not_found",
			args: args{
				ip: "127.0.0.1",
			},
			err: &APIError{
				HTTPCode: http.StatusNotFound,
				Message:  "no location found for the given IP address",
			},
		},
		{
			name: "found",
			args: args{ip: "70.95.73.73"},
			loc: &vio.Geolocation{
				IPAddress:   net.ParseIP("70.95.73.73"),
				CountryCode: "TL",
				Country:     "Saudi Arabia",
				City:        "Gradymouth",
				Latitude:    "-49.16675918861615",
				Longitude:   "-86.05920084416894",
				UpdatedAt:   time.Now(),
			},
		},
		{
			name: "foundIPv6",
			args: args{ip: "::ffff:70.95.73.73"},
			loc: &vio.Geolocation{
				IPAddress:   net.ParseIP("70.95.73.73"),
				CountryCode: "TL",
				Country:     "Saudi Arabia",
				City:        "Gradymouth",
				Latitude:    "-49.16675918861615",
				Longitude:   "-86.05920084416894",
				UpdatedAt:   time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := hs.Client().Get(hs.URL + "/lookup?ip=" + tt.args.ip)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			defer resp.Body.Close()
			dec := json.NewDecoder(resp.Body)

			if tt.err != nil {
				var gotErr *APIError
				if err := dec.Decode(&gotErr); err != nil {
					t.Errorf("cannot decode API error: %v", err)
				}
				if !cmp.Equal(tt.err, gotErr) {
					t.Errorf("Service.LookupLocation() error doesn't match: %v", cmp.Diff(tt.err, gotErr))
				}
				if gotErr.HTTPCode == http.StatusOK {
					t.Errorf("Service.LookupLocation() error HTTP code should not be 200")
				}
			} else {
				var got vio.Geolocation
				if err := dec.Decode(&got); err != nil {
					t.Errorf("cannot decode geolocation: %v", err)
				}
				if !cmp.Equal(tt.loc, &got, cmpopts.IgnoreFields(vio.Geolocation{}, "UpdatedAt")) {
					t.Errorf("Service.LookupLocation() doesn't match: %v", cmp.Diff(tt.loc, &got))
				}
			}
		})
	}
}

func FuzzLookup(f *testing.F) {
	// Restore DB sqltest's prefix.
	dbPrefix := sqltest.DatabasePrefix
	defer func() {
		sqltest.DatabasePrefix = dbPrefix
	}()
	sqltest.DatabasePrefix = "fuzz"

	migration := sqltest.New(f, sqltest.Options{
		Force:                   *force,
		Files:                   os.DirFS("../../migrations"),
		TemporaryDatabasePrefix: "fuzz_vio_api",
	})
	pool := migration.Setup(context.Background(), "")

	file, err := os.Open("../../testdata/example.csv")
	if err != nil {
		f.Fatal(err)
	}
	defer file.Close()

	stats, err := vio.NewImporter(3, slog.Default(), pool).Stream(context.Background(), file)
	if stats == nil {
		f.Error("stats should not be nil")
	}
	if err != nil {
		f.Errorf("cannot import location data: %v", err)
	}

	s := NewServer("", vio.NewService(vio.NewPostgres(pool, slog.Default())), slog.Default())
	hs := httptest.NewServer(http.HandlerFunc(s.lookupHandler))

	f.Add("127.0.0.1")
	f.Add("144.116.254.249")
	f.Add("73.178.254.104")
	f.Add("156.224.222.114")

	f.Fuzz(func(t *testing.T, ip string) {
		resp, err := hs.Client().Get(hs.URL + "/lookup?ip=" + ip)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()
	})
}
