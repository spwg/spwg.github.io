// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

func setupLogs() error {
	if _, err := os.Stat("./logs"); os.IsNotExist(err) {
		if err := os.Mkdir("./logs", 0777); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	errorLog, err := os.Create(fmt.Sprintf("./logs/error %s.log", now.Format(time.RFC822)))
	if err != nil {
		return err
	}
	serverLog, err := os.Create(fmt.Sprintf("./logs/gin %s.log", now.Format(time.RFC822)))
	if err != nil {
		return err
	}
	gin.DefaultWriter = io.MultiWriter(serverLog)
	gin.DefaultErrorWriter = io.MultiWriter(errorLog)
	return nil
}

func setupMiddleware(r *gin.Engine) {
	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:  []string{"spencerwgreene.com"},
		FrameDeny:     true,
		SSLRedirect:   true,
		SSLHost:       "localhost:8080",
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
	var forwardWWW gin.HandlerFunc = func(c *gin.Context) {
		if strings.HasPrefix(c.Request.Host, "www.") {
			to := "https://" + strings.TrimPrefix(c.Request.Host, "www.") + c.Request.URL.Path
			c.Redirect(http.StatusPermanentRedirect, to)
			c.Abort()
		}
	}
	r.Use(forwardWWW)
	r.Use(secureFunc)
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s\" \"%s\" \"%s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
}

func main() {
	if err := setupLogs(); err != nil {
		log.Fatalln(err)
	}
	r := gin.Default()
	setupMiddleware(r)
	r.Static("/", "./site")
	r.LoadHTMLFiles("./templates/404.tmpl")
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"path": c.Param("filepath"),
		})
	})
	if gin.IsDebugging() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalln(err)
		}
		return
	}
	go func() {
		if err := r.Run(":8081"); err != nil {
			log.Fatalln(err)
		}
	}()
	if err := r.RunTLS(
		":8080",
		"/etc/letsencrypt/live/spencerwgreene.com/fullchain.pem",
		"/etc/letsencrypt/live/spencerwgreene.com/privkey.pem"); err != nil {
		log.Fatalln(err)
	}
}
