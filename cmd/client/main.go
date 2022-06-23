package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yardbirdsax/to-sidecar-or-not/adder"
	"gopkg.in/square/go-jose.v2/json"
	"go.uber.org/zap"
)

var (
	clientResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "client_response_time",
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
		},
	)
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	r := gin.Default()
	serverAddr := os.Getenv("CLIENT_SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "server:8080"
	}
	serverURL := fmt.Sprintf("http://%s/adder", serverAddr)
	sidecarURL := fmt.Sprintf("http://%s/adder-with-sidecar", serverAddr)

	r.GET("/health", gin.WrapH(promhttp.Handler()))

	doHTTPAdder(serverURL, "adder-local", 10, 2)
	doHTTPAdder(sidecarURL, "adder-sidecar", 10, 2)

	r.Run()

}

func doHTTPAdder(URL string, route string, timeout int, delay int) {
	go func() {
		subLogger := zap.L().With(zap.String("name", route))
		for {
			time.Sleep(time.Duration(delay * int(time.Second)))
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout * int(time.Second)))
			defer cancel()
			input := &adder.AdderInput{
				First: 1,
				Second: 1,
			}
			requestJson, err := json.Marshal(input)
			if err != nil {
				subLogger.Error(err.Error())
				continue
			}
			requstBody := bytes.NewBuffer(requestJson)
			req, err := http.NewRequestWithContext(ctx, "POST", URL, requstBody)
			if err != nil {
				subLogger.Error(err.Error())
				continue
			}
			client := &http.Client{}
			timer := prometheus.NewTimer(clientResponseTime.WithLabelValues(route))
			resp, err := client.Do(req)
			if err != nil {
				subLogger.Error(err.Error())
				continue
			}
			duration := timer.ObserveDuration()
			subLogger.Info("end response", zap.Float64("duration", duration.Seconds()), zap.Int("status", resp.StatusCode))
		}
	}()
}