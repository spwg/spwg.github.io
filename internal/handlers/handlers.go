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
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template
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
	return s
}
