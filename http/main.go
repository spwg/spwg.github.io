// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"embed"
	"io/fs"
	"log"
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
		SSLRedirect:   true,
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
	if err := r.Run(":" + port); err != nil {
		log.Fatalln(err)
	}
}
