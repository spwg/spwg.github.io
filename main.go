// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

//go:embed cloudflare_ipv4.txt
var cloudflareIPv4Addresses string

//go:embed cloudflare_ipv6.txt
var cloudflareIPv6Addresses string

type requestCounter struct {
	count int
	mu    sync.Mutex
}

func (rc *requestCounter) Increment() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.count++
	fmt.Println(rc.count)
}

func (rc *requestCounter) Idle() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.count == 0
}

func prepare(r *gin.Engine) *requestCounter {
	// Setup logging and recovery first so that the logging happens first
	// and then recovery happens before any other middleware.
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	secureMiddleware := secure.New(secure.Options{
		AllowedHosts: []string{
			"spencergreene.com",
			"www.spencergreene.com",
			"spencergreene.fly.dev",
		},
		FrameDeny:         true,
		IsDevelopment:     gin.IsDebugging(),
		HostsProxyHeaders: []string{"X-Forwarded-Host"},
	})
	r.Use(func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}
		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	})
	r.SetTrustedProxies(append(strings.Fields(cloudflareIPv4Addresses), strings.Fields(cloudflareIPv6Addresses)...))
	rc := &requestCounter{}
	r.Use(func(c *gin.Context) {
		rc.Increment()
	})
	return rc
}

//go:embed site/*
var site embed.FS

func main() {
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if os.Getenv("FLY_APP_NAME") != "" {
		log.Printf("Running in the fly.io runtime.")
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	rc := prepare(engine)
	site, err := fs.Sub(site, "site")
	if err != nil {
		log.Fatal(err)
	}
	engine.GET("/", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, http.FS(site))
	})
	engine.GET("/api/dnschecker/:url", func(c *gin.Context) {
		// TODO: connect this to a page in site/ that polls this endpoint
		// for changes.
		ctx := c.Request.Context()
		u := c.Params.ByName("url")
		addrs, err := net.DefaultResolver.LookupHost(ctx, u)
		if err != nil {
			c.AbortWithStatus(500)
			return
		}
		b, err := json.Marshal(addrs)
		if err != nil {
			log.Print(err)
			c.AbortWithStatus(500)
			return
		}
		if _, err := c.Writer.Write(b); err != nil {
			log.Print(err)
			c.AbortWithStatus(500)
		}
	})
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: engine,
	}
	shutdown := make(chan bool)
	go func() {
		// Shutdown the server when idle. Fly will start it automatically when it receives a request.
		if gin.IsDebugging() {
			return
		}
		for {
			time.Sleep(time.Second)
			if rc.Idle() {
				log.Println("Connections are idle. Shutting down.")
				if err := srv.Shutdown(ctx); err != nil {
					log.Fatal(err)
				}
				shutdown <- true
				return
			}
		}
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	// Wait until Shutdown returns because ListenAndServe returns immediately when it's called.
	<-shutdown
}
