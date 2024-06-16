// Package database provides functionality for interacting with the database for the website.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

const (
	mostRecentFlightsQuery = "select distinct(flight_designator), max(seen_time) as most_recently_seen from flights group by flight_designator order by most_recently_seen desc limit $1;"
)

var (
	nycTZ = mustLoadLocation("America/New_York")
)

// FlightEntry is a row in the flights table.
type FlightEntry struct {
	// The flight designator.
	Code string `json:"code"`
	// Unix timestamp when the observation of this flight was made.
	WhenUnix int64 `json:"when"`
	// String representation of WhenUnix for display.
	When string
	// Time representation of WhenUnix.
	WhenTime time.Time
}

// Connection is a database connection.
type Connection struct {
	db       *sql.DB
	rowLimit int

	lastReloadMu sync.Mutex
	lastReload   time.Time

	allFlightsMu sync.Mutex
	allFlights   []*FlightEntry
}

// MostRecentFlights returns the most recent flights in the database according to the given row limit.
func (c *Connection) MostRecentFlights(ctx context.Context) ([]*FlightEntry, error) {
	c.allFlightsMu.Lock()
	defer c.allFlightsMu.Unlock()
	c.lastReloadMu.Lock()
	defer c.lastReloadMu.Unlock()
	if !c.lastReload.IsZero() && time.Since(c.lastReload) < time.Minute {
		return c.allFlights, nil
	}
	c.lastReload = time.Now()
	rows, err := c.db.QueryContext(ctx, mostRecentFlightsQuery, c.rowLimit)
	if err != nil {
		return nil, fmt.Errorf("loading most recent aircraft from fly postgres: query: %v", err)
	}
	defer rows.Close()
	var flights []*FlightEntry
	for rows.Next() {
		var code string
		var seen time.Time
		if err := rows.Scan(&code, &seen); err != nil {
			return nil, fmt.Errorf("loading most recent aircraft from fly postgres: scan: %v", err)
		}
		seen = seen.In(nycTZ)
		code = strings.TrimSpace(code)
		flights = append(flights, &FlightEntry{
			Code:     code,
			WhenUnix: seen.Unix(),
			When:     seen.Format("Jan 02, 2006 03:04:05 PM EST"),
			WhenTime: seen,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	glog.Infof("Loaded most recent flights.")
	c.allFlights = flights
	return flights, nil
}

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// Connect opens a new connection to the database at the given address.
func Connect(addr string) (*Connection, error) {
	glog.Infof("Connecting to the database.")
	db, err := sql.Open("pgx", addr)
	if err != nil {
		return nil, err
	}
	glog.Infof("Connected to the database.")
	conn := &Connection{
		db:       db,
		rowLimit: 100,
	}
	return conn, nil
}
