package vio

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrBadIPAddressFormat is returned when lookin up using a bad IP address format.
var ErrBadIPAddressFormat = fmt.Errorf("invalid IP address format")

// LookupLocation returns a location.
func (s *Service) LookupLocation(ctx context.Context, ip string) (*Geolocation, error) {
	addr := net.ParseIP(ip)
	if addr == nil {
		return nil, ErrBadIPAddressFormat
	}
	return s.db.LookupLocation(ctx, addr)
}

// NewService creates an API service.
func NewService(db DB) *Service {
	return &Service{db: db}
}

// Service for the API.
type Service struct {
	db DB
}

// DB layer.
//
//go:generate mockgen --build_flags=--mod=mod -package mock -destination internal/mock/mock.go . DB
type DB interface {
	// LookupLocation returns a location.
	LookupLocation(ctx context.Context, ip net.IP) (*Geolocation, error)
}

var _ DB = (*Postgres)(nil) // Check if methods expected by geolocation.DB are implemented correctly.

// ErrLocationNotFound is returned when a location is not found.
var ErrLocationNotFound = errors.New("location not found")
