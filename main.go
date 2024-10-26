// Binary main runs a web server for my (Spencer Greene's) personal website.
//
// It runs on the Fly app platform (fly.io) in 2 regions, San Jose and Atlanta.
// DNS is on Cloudflare, which also proxies requests through its CDN. There are
// A and AAAA records that point to the app on Fly. When a request is made for
// spencergreene.com, DNS resolution talks to the Cloudflare nameservers, which
// return IP addresses that route to its own CDN. If there's no cache hit, it
// sends the request to Fly. Fly terminates the TLS connection from Cloudflare
// and forwards the request to this web server.
package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/spwg/personal-website/internal/handlers"
	"github.com/unrolled/secure"
)

var (
	//go:embed cloudflare_ipv4.txt
	cloudflareIPv4Addresses string
	//go:embed cloudflare_ipv6.txt
	cloudflareIPv6Addresses string
	//go:embed static/*
	embeddedStatic embed.FS

	bindAddr = flag.String("bind_addr", defaultBindAddr(), "Full address to bind to.")
)

func defaultBindAddr() string {
	if os.Getenv("BIND_ADDR") != "" {
		return os.Getenv("BIND_ADDR")
	}
	return "localhost:8080"
}

// installMiddleware sets up logging and recovery first so that the logging
// happens first and then recovery happens before any other middleware.
func installMiddleware(r *gin.Engine) error {
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
	var proxies []string
	proxies = append(proxies, strings.Fields(cloudflareIPv4Addresses)...)
	proxies = append(proxies, strings.Fields(cloudflareIPv6Addresses)...)
	if err := r.SetTrustedProxies(proxies); err != nil {
		return fmt.Errorf("install middleware: %v", err)
	}
	return nil
}

func initializeSentry() (func(), error) {
	if os.Getenv("SENTRY_DSN") == "" {
		return func() {}, nil
	}
	glog.Infof("Initializing Sentry")
	options := sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	}
	if err := sentry.Init(options); err != nil {
		return func() {}, fmt.Errorf("sentry.Init: %v", err)
	}
	return func() { sentry.Flush(2 * time.Second) }, nil
}

func run() error {
	defer glog.Flush()
	flush, err := initializeSentry()
	if err != nil {
		return err
	}
	defer flush()
	engine := gin.New()
	installMiddleware(engine)
	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return err
	}
	_ = handlers.InstallRoutes(staticFS, engine)
	srv := &http.Server{
		Addr:    *bindAddr,
		Handler: engine,
	}
	glog.Infof("Listening on %q\n", *bindAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if err := flag.Set("alsologtostderr", "true"); err != nil {
		glog.Fatal(err)
	}
	if err := run(); err != nil {
		glog.Fatal(err)
	}
}
