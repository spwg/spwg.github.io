// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"log"
	"os"
	"io"

	"github.com/gin-gonic/gin"
)

func main() {
	f, err := os.Create("server.log")
	if err != nil {
		log.Fatalln(err)
	}
	gin.DefaultWriter = io.MultiWriter(f)
	r := gin.Default()
	r.Static("/", "./site")
	if err := r.Run(); err != nil {
		log.Fatalln(err)
	}
}
