// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net"
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

// DNSChecker is the endpoint for the dns tool.
func DNSChecker(t *template.Template) gin.HandlerFunc {
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
