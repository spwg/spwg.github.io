// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"html/template"
	"io/fs"
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spwg/personal-website/internal/database"
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template
	db     *database.Connection
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *Server) aircraftFeed(c *gin.Context) {
	flights, err := s.db.MostRecentFlights(c.Request.Context())
	if err != nil {
		glog.Error(err)
	}
	if err := s.t.Lookup("radar.tmpl").Execute(c.Writer, map[string]any{"Flights": flights}); err != nil {
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
func InstallRoutes(static fs.FS, engine *gin.Engine, db *database.Connection) *Server {
	t, err := template.ParseFS(static, "*.tmpl")
	if err != nil {
		glog.Fatal(err)
	}
	s := &Server{
		static: static,
		t:      t,
		db:     db,
	}
	engine.GET("/", s.root)
	engine.GET("/js/:path", s.js)
	engine.GET("/css/:path", s.css)
	engine.GET("/flights/radar/nyc", s.aircraftFeed)
	return s
}
