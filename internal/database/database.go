// Package database provides functionality for interacting with the database for the website.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

// MostRecentFlights returns the most recent flights in the database according to the given row limit.
func MostRecentFlights(ctx context.Context, db *sql.DB, rowLimit int) ([]*FlightEntry, error) {
	rows, err := db.QueryContext(ctx, mostRecentFlightsQuery, rowLimit)
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
	return flights, nil
}

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}
