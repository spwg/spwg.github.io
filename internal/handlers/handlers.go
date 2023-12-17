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

type dnsPage struct {
	Host        string
	IPAddresses []string
	NameServers []string
	NextReload  string
}

// dnsChecker is the endpoint for the dns tool.
func dnsChecker(t *template.Template) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		h, ok := c.GetQuery("host")
		if !ok {
			if err := t.Execute(c.Writer, map[string]any{"Result": dnsPage{}}); err != nil {
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
			err = t.ExecuteTemplate(&b, "dns_result", r)
		} else {
			// Go nested templates can only receive 1 argument.
			err = t.Execute(&b, map[string]any{"Result": r})
		}
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		if _, err := c.Writer.Write(b.Bytes()); err != nil {
			log.Print(err)
		}
	}
}

// AircraftFeed is the endpoint for aircraft data feed.
func AircraftFeed(c *gin.Context) {
	// TODO: configure the service to read from gcs, rate limiting, and making
	// the web page.
}

// InstallRoutes registers the server's routes on the given [*gin.Engine].
func InstallRoutes(sys fs.FS, engine *gin.Engine) *gin.Engine {
	installStaticRoutes(sys, engine)
	installDNSRoutes(sys, engine)
	return engine
}

// installStaticRoutes registers the routes for static pages on the engine.
func installStaticRoutes(staticFS fs.FS, engine *gin.Engine) {
	engine.GET("/", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, http.FS(staticFS))
	})
	engine.GET("/js/:path", func(c *gin.Context) {
		c.FileFromFS(path.Base(c.Request.URL.Path), http.FS(staticFS))
	})
	engine.GET("/css/:path", func(c *gin.Context) {
		c.FileFromFS(c.Params.ByName("path"), http.FS(staticFS))
	})
}

// installDNSRoutes registers the routes for the dns checker pages on the
// engine.
func installDNSRoutes(staticFS fs.FS, engine *gin.Engine) {
	t, err := template.ParseFS(staticFS, "dnschecker.tmpl", "dnsresult.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	engine.GET("/dnschecker", dnsChecker(t))
}
