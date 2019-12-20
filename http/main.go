// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"io"
	"log"
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
	gin.DefaultWriter = io.MultiWriter(serverLog, os.Stdout)
	gin.DefaultErrorWriter = io.MultiWriter(errorLog, os.Stderr)
	r := gin.Default()
	r.Static("/", "./site")
	if err := r.Run(); err != nil {
		log.Fatalln(err)
	}
}
