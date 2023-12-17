// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"bytes"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

type server struct {
	static fs.FS
	t      *template.Template
}

type dnsPage struct {
	Host        string
	IPAddresses []string
	NameServers []string
	NextReload  string
}

func (s *server) dnsChecker(c *gin.Context) {
	ctx := c.Request.Context()
	h, ok := c.GetQuery("host")
	if !ok {
		if err := s.t.Execute(c.Writer, map[string]any{"Result": dnsPage{}}); err != nil {
			c.Error(err)
		}
		return
	}
	lookupHostResponse, err := net.DefaultResolver.LookupHost(ctx, h)
	if err != nil {
		log.Print(err)
		switch err.(type) {
		case *net.DNSError:
		default:
			c.AbortWithError(500, err)
			return
		}
	}
	slices.Sort(lookupHostResponse)
	lookupNSResponse, err := net.DefaultResolver.LookupNS(ctx, h)
	if err != nil {
		log.Print(err)
		switch err.(type) {
		case *net.DNSError:
		default:
			c.AbortWithError(500, err)
			return
		}
	}
	slices.SortFunc(lookupNSResponse, func(a, b *net.NS) bool {
		return a.Host < b.Host
	})
	var nameServers []string
	for _, n := range lookupNSResponse {
		nameServers = append(nameServers, n.Host)
	}
	now := time.Now()
	r := dnsPage{
		Host:        h,
		IPAddresses: lookupHostResponse,
		NameServers: nameServers,
		NextReload:  now.UTC().Add(time.Minute).Format(time.RFC3339),
	}
	// This page uses htmx, which lets the server return html content to update just part of the page.
	var b bytes.Buffer
	if c.GetHeader("HX-Request") != "" {
		// Just execute return the HTML needed to update the page's result element.
		err = s.t.ExecuteTemplate(&b, "dns_result", r)
	} else {
		// Go nested templates can only receive 1 argument.
		err = s.t.Execute(&b, map[string]any{"Result": r})
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err := c.Writer.Write(b.Bytes()); err != nil {
		log.Print(err)
	}
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *server) aircraftFeed(c *gin.Context) {
	// TODO:gcs service configuration, rate limiting, the web page, and an integration test.
	//
	// For now, just display the stats.json file.
}

func (s *server) root(c *gin.Context) {
	c.FileFromFS(c.Request.URL.Path, http.FS(s.static))
}

func (s *server) js(c *gin.Context) {
	c.FileFromFS(path.Base(c.Request.URL.Path), http.FS(s.static))
}

func (s *server) css(c *gin.Context) {
	c.FileFromFS(c.Params.ByName("path"), http.FS(s.static))
}

// InstallRoutes registers the server's routes on the given [*gin.Engine].
func InstallRoutes(static fs.FS, engine *gin.Engine) *gin.Engine {
	t, err := template.ParseFS(static, "dnschecker.tmpl", "dnsresult.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	s := &server{static, t}
	engine.GET("/", s.root)
	engine.GET("/js/:path", s.js)
	engine.GET("/css/:path", s.css)
	engine.GET("/dnschecker", s.dnsChecker)
	engine.GET("/aircraft/feed", s.aircraftFeed)
	return engine
}
