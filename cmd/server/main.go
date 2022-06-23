package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yardbirdsax/to-sidecar-or-not/adder"
)

var (
	serverResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "server_response_time",
			Buckets: []float64{
				1,
				5,
				10,
				20,
				100,
			},
		},
		[]string{
			"route",
			"stage",
		},
	)
)

func main() {
	r := gin.Default()
	r.GET("/health", gin.WrapH(promhttp.Handler()))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/adder-with-sidecar", func(c *gin.Context) {
		routeName := "adder-with-sidecar"
		totalTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "total"}))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/adder", "http://localhost:8081"), c.Request.Body)
		if err != nil {
			c.AbortWithError(500, err)
		}
		client := http.Client{}
		remoteTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "remote"}))
		resp, err := client.Do(req)
		if err != nil {
			c.AbortWithError(500, err)
		}
		respBodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.AbortWithError(500, err)
		}
		remoteDuration := remoteTimer.ObserveDuration().Seconds()
		c.JSON(200, gin.H{
			"resp": string(respBodyBytes),
		})
		duration := totalTimer.ObserveDuration().Seconds()
		localDuration := duration - remoteDuration
		serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "local"}).Observe(localDuration)
	})
	adder := &adder.Adder{
		ServerResponsePromClient: serverResponseTime,
	}
	r.POST("/adder", adder.ServeHTTP)
	r.Run()
}