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

func main() {
	if err := setupLogs(); err != nil {
		log.Fatalln(err)
	}
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
