package vio

import (
	"encoding/json"
	"net"
	"time"
)

// Geolocation data.
type Geolocation struct {
	IPAddress   net.IP
	CountryCode string
	Country     string
	City        string
	Latitude    json.Number
	Longitude   json.Number
	UpdatedAt   time.Time
}
