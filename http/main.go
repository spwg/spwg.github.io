// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	serverLog, err := os.Create("server.log")
	if err != nil {
		log.Fatalln(err)
	}
	errorLog, err := os.Create("error.log")
	if err != nil {
		log.Fatalln(err)
	}
	gin.DefaultWriter = io.MultiWriter(serverLog)
	gin.DefaultErrorWriter = io.MultiWriter(errorLog)
	r := gin.Default()
	r.Static("/", "./site")
	r.LoadHTMLFiles("./templates/404.tmpl")
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"path": c.Param("filepath"),
		})
	})
	if err := r.Run(); err != nil {
		log.Fatalln(err)
	}
}
