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

	gcsReadRate = flag.Duration("gcs_read_rate", 6*time.Hour,
		"Frequency at which to read from google cloud storage.")
	downloadHistoricalDataFromGCS = flag.Bool("download_historical_data_from_gcs", os.Getenv("FLY_APP_NAME") != "",
		"Enable background tasks such as reading from GCS.")
	dump1090DataDirectory = flag.String("dump1090_data_directory", ".",
		"If in dev, read from this directory instead of GCS.")
	useDataDirectory = flag.Bool("use_data_directory", os.Getenv("FLY_APP_NAME") == "", "")
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
	server := handlers.InstallRoutes(staticFS, engine, *gcsReadRate)
	if *useDataDirectory && !*downloadHistoricalDataFromGCS {
		if err := server.SetDump1090DataDirectory(*dump1090DataDirectory); err != nil {
			return err
		}
	}
	if *downloadHistoricalDataFromGCS {
		go func() {
			l := rate.NewLimiter(rate.Every(time.Hour), 1)
			for {
				if err := l.Wait(ctx); err != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err := server.DownloadAllAircraftFileFromGCS(ctx); err != nil {
					glog.Fatal(err)
				}
			}
		}()
	}
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
