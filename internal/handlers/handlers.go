// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"golang.org/x/time/rate"
)

var nycTZ = mustLoadLocation("America/New_York")

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template

	allFlightsMu sync.Mutex
	allFlights   []*flightEntry

	db          *sql.DB
	reloadLimit *rate.Limiter
	queryLimit  int
}

type flightEntry struct {
	Code     string `json:"code"`
	WhenUnix int64  `json:"when"`
	When     string
	WhenTime time.Time
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *Server) aircraftFeed(c *gin.Context) {
	if err := s.loadMostRecentAircraftFromFlyPostgres(c.Request.Context()); err != nil {
		glog.Error(err)
	}
	s.allFlightsMu.Lock()
	defer s.allFlightsMu.Unlock()
	if err := s.t.Lookup("radar.tmpl").Execute(c.Writer, map[string]any{"Flights": s.allFlights}); err != nil {
		c.Error(err)
		return
	}
}

func (s *Server) loadMostRecentAircraftFromFlyPostgres(ctx context.Context) error {
	if !s.reloadLimit.Reserve().OK() {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, "select distinct(flight_designator), max(seen_time) as most_recently_seen from flights group by flight_designator order by most_recently_seen desc limit $1;", s.queryLimit)
	if err != nil {
		return fmt.Errorf("loading most recent aircraft from fly postgres: query: %v", err)
	}
	defer rows.Close()
	var flights []*flightEntry
	for rows.Next() {
		var code string
		var seen time.Time
		if err := rows.Scan(&code, &seen); err != nil {
			return fmt.Errorf("loading most recent aircraft from fly postgres: scan: %v", err)
		}
		seen = seen.In(nycTZ)
		code = strings.TrimSpace(code)
		flights = append(flights, &flightEntry{
			Code:     code,
			WhenUnix: seen.Unix(),
			When:     seen.Format("Jan 02, 2006 03:04:05 PM EST"),
			WhenTime: seen,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.allFlightsMu.Lock()
	s.allFlights = flights
	s.allFlightsMu.Unlock()
	glog.Infof("Loaded most recent flights.")
	return nil
}

func (s *Server) root(c *gin.Context) {
	c.FileFromFS(c.Request.URL.Path, http.FS(s.static))
}

func (s *Server) js(c *gin.Context) {
	c.FileFromFS(path.Base(c.Request.URL.Path), http.FS(s.static))
}

func (s *Server) css(c *gin.Context) {
	c.FileFromFS(c.Params.ByName("path"), http.FS(s.static))
}

// InstallRoutes registers the server's routes on the given [*gin.Engine].
func InstallRoutes(static fs.FS, engine *gin.Engine, db *sql.DB, reloadLimit *rate.Limiter, limit int) *Server {
	t, err := template.ParseFS(static, "*.tmpl")
	if err != nil {
		glog.Fatal(err)
	}
	s := &Server{
		static:      static,
		t:           t,
		db:          db,
		reloadLimit: reloadLimit,
		queryLimit:  limit,
	}
	engine.GET("/", s.root)
	engine.GET("/js/:path", s.js)
	engine.GET("/css/:path", s.css)
	engine.GET("/flights/radar/nyc", s.aircraftFeed)
	return s
}

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}
