package vio_test

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/henvic/pgtools/sqltest"
	"github.com/henvic/vio"
	"github.com/henvic/vio/internal/mock"
	"go.uber.org/mock/gomock"
)

var force = flag.Bool("force", false, "Force cleaning the database before starting")

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION_TESTDB") != "true" {
		log.Printf("Skipping tests that require database connection")
		return
	}
	os.Exit(m.Run())
}

func TestServiceLookupLocation(t *testing.T) {
	t.Parallel()
	migration := sqltest.New(t, sqltest.Options{
		Force: *force,
		Files: os.DirFS("migrations"),
	})
	pool := migration.Setup(context.Background(), "")

	service := vio.NewService(vio.NewPostgres(pool, slog.Default()))

	file, err := os.Open("testdata/example.csv")
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

	type args struct {
		ctx context.Context
		ip  string
	}
	tests := []struct {
		name    string
		args    args
		mock    func(t testing.TB) *mock.MockDB // Leave as nil for using a real database implementation.
		want    *vio.Geolocation
		wantErr string
	}{
		{
			name: "empty_ip",
			args: args{
				ctx: context.Background(),
				ip:  "",
			},
			wantErr: "invalid IP address format",
		},
		{
			name: "invalid_ip",
			args: args{
				ctx: context.Background(),
				ip:  "123",
			},
			wantErr: "invalid IP address format",
		},
		{
			name: "ip1",
			args: args{
				ctx: context.Background(),
				ip:  "160.103.7.140",
			},
			want: &vio.Geolocation{
				IPAddress:   net.ParseIP("160.103.7.140"),
				CountryCode: "CZ",
				Country:     "Nicaragua",
				City:        "New Neva",
				Latitude:    "-68.31023296602508",
				Longitude:   "-37.62435199624531",
				UpdatedAt:   time.Now(),
			},
		},
		{
			name: "ip2",
			args: args{
				ctx: context.Background(),
				ip:  "125.159.20.54",
			},
			want: &vio.Geolocation{
				IPAddress:   net.ParseIP("125.159.20.54"),
				CountryCode: "LI",
				Country:     "Guyana",
				City:        "Port Karson",
				Latitude:    "-78.2274228596799",
				Longitude:   "-163.26218895343357",
				UpdatedAt:   time.Now(),
			},
		},
		{
			name: "ip2v6",
			args: args{
				ctx: context.Background(),
				ip:  "::ffff:125.159.20.54",
			},
			want: &vio.Geolocation{
				IPAddress:   net.ParseIP("125.159.20.54"),
				CountryCode: "LI",
				Country:     "Guyana",
				City:        "Port Karson",
				Latitude:    "-78.2274228596799",
				Longitude:   "-163.26218895343357",
				UpdatedAt:   time.Now(),
			},
		},
		{
			name: "not_found",
			args: args{
				ctx: context.Background(),
				ip:  "127.0.0.1",
			},
			want: nil,
		},
		{
			name: "canceled_ctx",
			args: args{
				ctx: canceledContext(),
				ip:  "127.0.0.1",
			},
			wantErr: "context canceled",
		},
		{
			name: "deadline_exceeded_ctx",
			args: args{
				ctx: deadlineExceededContext(),
				ip:  "127.0.0.1",
			},
			wantErr: "context deadline exceeded",
		},
		{
			name: "database_error",
			args: args{
				ctx: context.Background(),
				ip:  "127.0.0.1",
			},
			mock: func(t testing.TB) *mock.MockDB {
				ctrl := gomock.NewController(t)
				m := mock.NewMockDB(ctrl)
				m.EXPECT().LookupLocation(gomock.Not(gomock.Nil()), net.ParseIP("127.0.0.1")).Return(nil, errors.New("unexpected error"))
				return m
			},
			wantErr: "unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If tt.mock is nil, use real database implementation if available. Otherwise, skip the test.
			var s = service
			if tt.mock != nil {
				s = vio.NewService(tt.mock(t))
			} else if s == nil {
				t.Skip("required database not found, skipping test")
			}
			got, err := s.LookupLocation(tt.args.ctx, tt.args.ip)
			if err == nil && tt.wantErr != "" || err != nil && tt.wantErr != err.Error() {
				t.Errorf("Service.LookupLocation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !cmp.Equal(tt.want, got, cmpopts.EquateApproxTime(time.Minute)) {
				t.Errorf("value returned by Service.LookupLocation() doesn't match: %v", cmp.Diff(tt.want, got))
			}
		})
	}
}
