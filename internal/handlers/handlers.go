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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"golang.org/x/time/rate"
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template

	allFlights []*flightEntry
}

type flightEntry struct {
	Code     string `json:"code"`
	WhenUnix int64  `json:"when"`
	When     string
	WhenTime time.Time
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *Server) aircraftFeed(c *gin.Context) {
	if err := s.t.Lookup("radar.tmpl").Execute(c.Writer, map[string]any{"Flights": s.allFlights}); err != nil {
		c.Error(err)
		return
	}
}

func (s *Server) LoadMostRecentAircraftFromFlyPostgres(ctx context.Context, db *sql.DB, reloadLimit *rate.Limiter, limit int) error {
	for {
		if err := reloadLimit.Wait(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		rows, err := db.Query("select distinct(flight_designator), max(seen_time) as most_recently_seen from flights group by flight_designator order by most_recently_seen desc limit $1;", limit)
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
			code = strings.TrimSpace(code)
			flights = append(flights, &flightEntry{
				Code:     code,
				WhenUnix: seen.Unix(),
				When:     seen.Format(time.ANSIC),
				WhenTime: seen,
			})
		}
		s.allFlights = flights
	}
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
func InstallRoutes(static fs.FS, engine *gin.Engine) *Server {
	t, err := template.ParseFS(static, "*.tmpl")
	if err != nil {
		glog.Fatal(err)
	}
	s := &Server{
		static: static,
		t:      t,
	}
	engine.GET("/", s.root)
	engine.GET("/js/:path", s.js)
	engine.GET("/css/:path", s.css)
	engine.GET("/flights/radar/nyc", s.aircraftFeed)
	return s
}
