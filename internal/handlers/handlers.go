// Package handlers provides functionality for endpoints that should be
// installed on the server.
package handlers

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"golang.org/x/exp/slices"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Server holds a collection of service endpoints.
type Server struct {
	static fs.FS
	t      *template.Template

	historicalRadarData map[string]*historicalRadarEntry
	allFlights          []*flightEntry
}

type historicalRadarEntry struct {
	Now      float32       `json:"now"` // unix seconds
	Aircraft []flightEntry `json:"aircraft"`
}

type flightEntry struct {
	Code     string `json:"code"`
	WhenUnix int64  `json:"when"`
	When     string
	WhenTime time.Time
}

// aircraftFeed is the endpoint for aircraft data feed.
func (s *Server) aircraftFeed(c *gin.Context) {
	if err := s.t.Lookup("radar.tmpl").Execute(c.Writer, map[string]any{"Flights": s.allFlights}); err != nil {
		c.Error(err)
		return
	}
}

func (s *Server) SetDump1090DataDirectory(dir string) error {
	allAircraftPath := path.Join(dir, "all_aircraft.json")
	b, err := os.ReadFile(allAircraftPath)
	if err != nil {
		return err
	}
	flights, err := unmarshalJSONAllAircraft(b)
	if err != nil {
		return err
	}
	s.allFlights = mostRecentTimestamps(flights)
	glog.Infof("Loaded all aircraft from %v\n", allAircraftPath)
	return nil
}

// MostRecentTimestamps creates a new slices of flight entries that contains
// only 1 entry per flight code in the input flights.
//
// The returned slice will be sorted by timestamp (most recent first).
func mostRecentTimestamps(flights []*flightEntry) []*flightEntry {
	if len(flights) == 0 {
		return nil
	}
	slices.SortStableFunc(flights, func(a, b *flightEntry) bool {
		x := or(cmp.Compare(a.Code, b.Code), cmp.Compare(a.WhenUnix, b.WhenUnix))
		return x < 0
	})
	filtered := []*flightEntry{flights[0]}
	for _, f := range flights[1:] {
		if filtered[len(filtered)-1].Code == f.Code {
			continue
		}
		filtered = append(filtered, f)
	}
	slices.SortStableFunc(filtered, func(a, b *flightEntry) bool {
		return !cmp.Less(a.WhenUnix, b.WhenUnix)
	})
	return filtered
}

func unmarshalJSONAllAircraft(b []byte) ([]*flightEntry, error) {
	var flights []*flightEntry
	if err := json.Unmarshal(b, &flights); err != nil {
		return nil, err
	}
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	for _, a := range flights {
		a.WhenTime = time.Unix(a.WhenUnix, 0).In(nyc)
		a.When = a.WhenTime.Format(time.UnixDate)
	}
	return flights, nil
}

// DownloadHistoricalDataFromGCS loads historical files from GCS.
func (s *Server) DownloadHistoricalDataFromGCS(ctx context.Context) error {
	// When GOOGLE_APPLICATION_CREDENTIALS_JSON is set, it'll be the JSON
	// contents of the credentials necessary to authenticate with Google Cloud.
	// This is currently configured with Fly Secrets, which are available to the
	// server at run time only. Docs: https://fly.io/docs/reference/secrets/.
	options := gcsClientOptions()
	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		return err
	}
	defer client.Close()
	m, err := loadHistoricalRadarData(ctx, client)
	if err != nil {
		return err
	}
	s.historicalRadarData = m
	glog.Infof("Loaded radar data from GCS.\n")
	return nil
}

func (s *Server) DownloadAllAircraftFileFromGCS(ctx context.Context) error {
	options := gcsClientOptions()
	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		return err
	}
	defer client.Close()
	flights, err := loadAllAircraftFile(ctx, client)
	if err != nil {
		return err
	}
	s.allFlights = mostRecentTimestamps(flights)
	glog.Infof("Loaded all aircraft from GCS.\n")
	return nil
}

func (s *Server) LoadMostRecentAircraftFromFlyPostgres(ctx context.Context, db *sql.DB, limit int) error {
	rows, err := db.Query("select distinct(flight_designator), max(seen_time) as most_recently_seen from flights group by flight_designator order by most_recently_seen desc limit $1;", limit)
	if err != nil {
		return fmt.Errorf("loading most recent aircraft from fly postgres: query: %v", err)
	}
	defer rows.Close()
	var allFlights []*flightEntry
	for rows.Next() {
		var code string
		var seen time.Time
		if err := rows.Scan(&code, &seen); err != nil {
			return fmt.Errorf("loading most recent aircraft from fly postgres: scan: %v", err)
		}
		allFlights = append(allFlights, &flightEntry{
			Code:     code,
			WhenUnix: seen.Unix(),
			When:     seen.Format(time.ANSIC),
			WhenTime: seen,
		})
	}
	s.allFlights = allFlights
	return nil
}

func gcsClientOptions() []option.ClientOption {
	var options []option.ClientOption
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		glog.Infoln("Using credentials from GOOGLE_APPLICATION_CREDENTIALS.")
		options = append(options, option.WithCredentialsJSON([]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))))
	} else {
		glog.Infoln("No credentials found in environment variables, this is fine in local dev.")
	}
	return options
}

func loadAllAircraftFile(ctx context.Context, client *storage.Client) ([]*flightEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	bucket := client.Bucket("dump1090-data")
	r, err := bucket.Object("all_aircraft.json").NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return unmarshalJSONAllAircraft(b)
}

func loadHistoricalRadarData(ctx context.Context, client *storage.Client) (map[string]*historicalRadarEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	bucket := client.Bucket("dump1090-data")
	itr := bucket.Objects(ctx, &storage.Query{
		Prefix: "history",
	})
	m := map[string]*historicalRadarEntry{}
	for {
		attrs, err := itr.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, err
		}
		r, err := bucket.Object(attrs.Name).NewReader(ctx)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		glog.Infoln("Download", attrs.Name)
		b, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		entry := &historicalRadarEntry{}
		if err := json.Unmarshal(b, entry); err != nil {
			return nil, err
		}
		m[attrs.Name] = entry
	}
	glog.Infoln("Download complete")
	return m, nil
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

func or[T cmp.Ordered](args ...T) T {
	var zero T
	for _, a := range args {
		if a != zero {
			return a
		}
	}
	return zero
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
	engine.GET("/flights/radar/nyc", s.aircraftFeed)
	return s
}
