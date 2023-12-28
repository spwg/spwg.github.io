// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template

	// gcsReadLimiter rate limits reads from google cloud storage.
	gcsReadLimiter *rate.Limiter

	// stats is the most recently read bytes of the stats.json file.
	statsMu sync.Mutex
	stats   []byte

	// ready is closed when the server is ready to start.
	ready     chan struct{}
	readyOnce sync.Once
}

type dnsPage struct {
	Host        string
	IPAddresses []string
	NameServers []string
	NextReload  string
}

func (s *Server) dnsChecker(c *gin.Context) {
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
func (s *Server) aircraftFeed(c *gin.Context) {
	// TODO: the web page and an integration test.
	//
	// For now, just display the stats.json file.
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	if _, err := c.Writer.Write(s.stats); err != nil {
		c.Error(err)
	}
}

// RunBackgroundTasks runs tasks until ctx is canceled or an error is
// encountered. Should be called in a goroutine.
func (s *Server) RunBackgroundTasks(ctx context.Context) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	done := ctx.Done()
	for {
		select {
		case <-done:
			return nil
		default:
		}
		log.Println("Waiting")
		if err := s.gcsReadLimiter.Wait(ctx); err != nil {
			return err
		}
		log.Println("Downloading stats.json file")
		b, err := downloadStats(ctx, client)
		if err != nil {
			return err
		}
		log.Println("Download complete")
		s.statsMu.Lock()
		s.stats = b
		s.statsMu.Unlock()
		s.readyOnce.Do(func() {
			close(s.ready)
		})
	}
}

func downloadStats(ctx context.Context, client *storage.Client) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	r, err := client.Bucket("dump1090-data").Object("stats.json").NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, buf.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	return pretty.Bytes(), nil
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

// Ready waits until server initialization is done.
func (s *Server) Ready(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InstallRoutes registers the server's routes on the given [*gin.Engine].
func InstallRoutes(static fs.FS, engine *gin.Engine, statsRefresh time.Duration) *Server {
	t, err := template.ParseFS(static, "dnschecker.tmpl", "dnsresult.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	s := &Server{
		static:         static,
		t:              t,
		gcsReadLimiter: rate.NewLimiter(rate.Every(statsRefresh), 1),
		statsMu:        sync.Mutex{},
		ready:          make(chan struct{}),
	}
	engine.GET("/", s.root)
	engine.GET("/js/:path", s.js)
	engine.GET("/css/:path", s.css)
	engine.GET("/dnschecker", s.dnsChecker)
	engine.GET("/aircraft/feed", s.aircraftFeed)
	return s
}
