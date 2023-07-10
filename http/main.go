// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

func setupMiddleware(r *gin.Engine) {
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
		FrameDeny:     true,
		IsDevelopment: gin.IsDebugging(),
	})
	var secureFunc gin.HandlerFunc = func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}
		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	}
	r.Use(secureFunc)
}

//go:embed site/*
var site embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if os.Getenv("FLY_APP_NAME") != "" {
		log.Printf("Running in the fly.io runtime.")
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	setupMiddleware(r)
	site, err := fs.Sub(site, "site")
	if err != nil {
		log.Fatal(err)
	}
	r.GET("/", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, http.FS(site))
	})
	r.GET("/api/dnschecker/:url", func(c *gin.Context) {
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
	if err := r.Run(":" + port); err != nil {
		log.Fatalln(err)
	}
}
