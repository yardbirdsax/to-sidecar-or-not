package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yardbirdsax/to-sidecar-or-not/adder"
)

func main() {
	upstreamAddr := os.Getenv("SIDECAR_UPSTREAM_ADDR")
	if upstreamAddr == "" {
		upstreamAddr = "localhost:8080"
	}
	upstreamURL := fmt.Sprintf("http://%s", upstreamAddr)
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/ping", upstreamURL), nil)
		if err != nil {
			c.AbortWithError(500, err)
		}
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.AbortWithError(500, err)
		}
		respBodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.AbortWithError(500, err)
		}
		c.JSON(200, gin.H{
			"resp": string(respBodyBytes),
		})
	})
	adder := &adder.Adder{}
	r.POST("/adder", adder.ServeHTTP)
	r.Run()
}
