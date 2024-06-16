// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spwg/personal-website/internal/database"
	"golang.org/x/time/rate"
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template

	allFlightsMu sync.Mutex
	allFlights   []*database.FlightEntry

	db          *sql.DB
	reloadLimit *rate.Limiter
	queryLimit  int
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *Server) aircraftFeed(c *gin.Context) {
	if s.reloadLimit.Reserve().OK() {
		flights, err := database.MostRecentFlights(c.Request.Context(), s.db, s.queryLimit)
		if err != nil {
			glog.Error(err)
		}
		s.allFlightsMu.Lock()
		s.allFlights = flights
		defer s.allFlightsMu.Unlock()
	}
	if err := s.t.Lookup("radar.tmpl").Execute(c.Writer, map[string]any{"Flights": s.allFlights}); err != nil {
		c.Error(err)
		return
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
