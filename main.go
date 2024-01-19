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
	"context"
	"database/sql"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
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
	"golang.org/x/time/rate"
)

var (
	//go:embed cloudflare_ipv4.txt
	cloudflareIPv4Addresses string
	//go:embed cloudflare_ipv6.txt
	cloudflareIPv6Addresses string
	//go:embed static/*
	embeddedStatic embed.FS
)

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

func run(ctx context.Context) error {
	defer glog.Flush()
	if os.Getenv("SENTRY_DSN") != "" {
		glog.Infof("Initializing Sentry")
		options := sentry.ClientOptions{
			Dsn:              os.Getenv("SENTRY_DSN"),
			TracesSampleRate: 1.0,
		}
		if err := sentry.Init(options); err != nil {
			return fmt.Errorf("sentry.Init: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	var host string
	if os.Getenv("FLY_APP_NAME") != "" {
		glog.Infof("Running in the fly.io runtime.")
		gin.SetMode(gin.ReleaseMode)
		host = "::"
	} else {
		host = "::1"
	}
	engine := gin.New()
	installMiddleware(engine)
	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return err
	}
	glog.Infof("Connecting to the database.")
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	glog.Infof("Connected to the database.")
	reloadLimit := rate.NewLimiter(rate.Every(2*time.Minute), 1)
	rowLimit := 100
	_ = handlers.InstallRoutes(staticFS, engine, db, reloadLimit, rowLimit)
	srv := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: engine,
	}
	glog.Infof("Listening on %v\n", net.JoinHostPort(host, port))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	ctx := context.Background()
	flag.Parse()
	if err := flag.Set("alsologtostderr", "true"); err != nil {
		glog.Fatal(err)
	}
	if err := run(ctx); err != nil {
		glog.Fatal(err)
	}
}
