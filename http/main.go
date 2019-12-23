// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

func setupLogs() error {
	if _, err := os.Stat("./logs"); os.IsNotExist(err) {
		if err := os.Mkdir("./logs", 0777); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	serverLog, err := os.Create(fmt.Sprintf("./logs/gin %s.log", now.Format(time.RFC822)))
	if err != nil {
		return err
	}
	errorLog, err := os.Create(fmt.Sprintf("./logs/error %s.log", now.Format(time.RFC822)))
	if err != nil {
		return err
	}
	gin.DefaultWriter = io.MultiWriter(serverLog)
	gin.DefaultErrorWriter = io.MultiWriter(errorLog)
	return nil
}

func routerSWG() http.Handler {
	r := gin.Default()
	r.Static("/", "./site")
	r.LoadHTMLFiles("./templates/404.tmpl")
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"path": c.Param("filepath"),
		})
	})
	return r
}

func routerGreyskull() http.Handler {
	r := gin.Default()
	r.Static("/greyskull", "./site/greyskull")
	return r
}

func main() {
	if err := setupLogs(); err != nil {
		log.Fatalln(err)
	}
	var g errgroup.Group
	swgSrv := &http.Server{
		Addr:         ":8080",
		Handler:      routerSWG(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	greyskullSrv := &http.Server{
		Addr:         ":8081",
		Handler:      routerGreyskull(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	g.Go(func() error {
		if gin.IsDebugging() {
			return swgSrv.ListenAndServe()
		}
		return swgSrv.ListenAndServeTLS(
			"/etc/letsencrypt/live/spencerwgreene.com/fullchain.pem",
			"/etc/letsencrypt/live/spencerwgreene.com/privkey.pem")
	})
	g.Go(func() error {
		return greyskullSrv.ListenAndServe()
	})
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
