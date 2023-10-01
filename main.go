// Binary main runs a web server for my (Spencer Greene's) personal website.
//
// It runs on the Fly app platform (fly.io) in 2 regions, San Jose and Atlanta.
// DNS is on Cloudflare, which also proxies requests through its CDN. There are
// A and AAAA records that point to the app on Fly. When a request is made for
// spencergreene.com, DNS resolution talks to the Cloudflare nameservers, which
// return IP addresses that route to its own CDN. If there's no cache hit, it
// sends the request to Fly. Fly terminates the TLS connection from Cloudflare
// and forwards the request to this web server.
//
// If necessary, Fly will start the server to respond to the request. That means
// this binary should start up fast. To suspend again, it exits after a period
// of idleness, which means processing zero requests, but only if the flag
// --shutdown_on_idle is true. So, most of the time it doesn't consume any CPU
// or memory because it's not running.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
	"golang.org/x/exp/slices"
)

var (
	//go:embed cloudflare_ipv4.txt
	cloudflareIPv4Addresses string
	//go:embed cloudflare_ipv6.txt
	cloudflareIPv6Addresses string
	//go:embed static/*
	embeddedStatic embed.FS

	shutdownOnIdle = flag.Bool("shutdown_on_idle", false, "Whether to exit after a period of idleness.")
)

type dnsPage struct {
	Host        string
	IPAddresses []string
	NameServers []string
	NextReload  string
}

type requestCounter struct {
	processing int
	last       time.Time
	mu         sync.Mutex
}

func (rc *requestCounter) Increment() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.last = time.Now()
	rc.processing++
}

func (rc *requestCounter) Decrement() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.last = time.Now()
	rc.processing--
}

// Idle returns whether the server is idle and can suspend.
func (rc *requestCounter) Idle() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.processing == 0 && time.Since(rc.last) > time.Second
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
		c.Next()
		rc.Decrement()
	})
	return rc
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()
	ctx := context.Background()
	if os.Getenv("SENTRY_DSN") != "" {
		log.Printf("Initializing Sentry")
		options := sentry.ClientOptions{
			Dsn:              os.Getenv("SENTRY_DSN"),
			TracesSampleRate: 1.0,
		}
		if err := sentry.Init(options); err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	var host string
	if os.Getenv("FLY_APP_NAME") != "" {
		log.Printf("Running in the fly.io runtime.")
		gin.SetMode(gin.ReleaseMode)
		host = "::"
	} else {
		host = "::1"
	}
	engine := gin.New()
	rc := prepare(engine)
	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		log.Fatal(err)
	}
	engine.GET("/", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, http.FS(staticFS))
	})
	engine.GET("/js/:path", func(c *gin.Context) {
		c.FileFromFS(path.Base(c.Request.URL.Path), http.FS(staticFS))
	})
	engine.GET("/css/:path", func(c *gin.Context) {
		c.FileFromFS(c.Params.ByName("path"), http.FS(staticFS))
	})

	t, err := template.ParseFS(staticFS, "dnschecker.tmpl", "dnsresult.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	engine.GET("/dnschecker", func(c *gin.Context) {
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
	})
	srv := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	// Wait until Shutdown returns because ListenAndServe returns immediately when it's called.
	// Shutdown the server when idle. Fly will start it automatically when it receives a request.
	for {
		// Wait a minute before shutdown in order to let Fly health check the server first
		// exiting. This helps with health checks during deployments.
		time.Sleep(time.Minute)
		if gin.IsDebugging() {
			continue
		}
		if !*shutdownOnIdle {
			continue
		}
		if rc.Idle() {
			log.Println("Connections are idle. Shutting down.")
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Fatal(err)
			}
			return
		}
	}
}
