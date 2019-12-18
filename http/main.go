// Binary webhttps runs a web server that servers spencerwgreene.com.
package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Static("/", "./site")
	if err := r.Run(); err != nil {
		log.Fatalln(err)
	}
}
