package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ginpprof "github.com/gin-contrib/pprof"
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yardbirdsax/to-sidecar-or-not/adder"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	// We use pprof (https://pkg.go.dev/runtime/pprof) for tracking the time spent on each call.
	if err != nil {
		zap.L().Panic(err.Error())
	}
	

	// Handle interrupt signals
	stopChan := make(chan bool, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func(){
		zap.L().Debug("starting interrupt goroutine")
		signal := <- sigChan
		zap.L().Info(fmt.Sprintf("received signal: %s", signal))
		close(stopChan)
	}()

	// Setup gRPC clients for the sidecar; one uses Unix sockets, one HTTP.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sockAddr := os.Getenv("SOCK_ADDR")
	if sockAddr == "" {
		zap.L().Panic("required env variable SOCK_ADDR is not set")
	}
	unixConn, err := grpc.DialContext(ctx, fmt.Sprintf("unix://%s", sockAddr), grpc.WithInsecure())
	if err != nil {
		zap.L().Panic(err.Error())
	}
	gRPCClientSocket := adder.NewAdderClient(unixConn)
	conn, err := grpc.DialContext(ctx, "127.0.0.1:8082", grpc.WithInsecure())
	if err != nil {
		zap.L().Panic(err.Error())
	}
	gRPCClient := adder.NewAdderClient(conn)
	
	r := gin.New()
	ginpprof.Register(r)
	r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))
	r.GET("/health", gin.WrapH(promhttp.Handler()))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	// This endpoint uses a REST call over HTTP to connect to the sidecar
	r.POST("/adder-with-sidecar", func(c *gin.Context) {
		routeName := "adder-with-sidecar"
		totalTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "total"}))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		calcTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "calc"}))
		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/adder", "http://localhost:8081"), c.Request.Body)
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
		response := &adder.AdderResponse{}
		err = json.Unmarshal(respBodyBytes, response)
		if err != nil {
			c.AbortWithError(500, err)
		}
		calcTimer.ObserveDuration()

		c.JSON(200, response)
		totalTimer.ObserveDuration()
	})
	// This endpoint uses a gRPC call over HTTP to connect to the sidecar
	r.POST("/adder-with-sidecar-grpc", func(c *gin.Context) {
		routeName := "adder-with-grpc-sidecar"
		totalTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "total"}))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		body := &adder.AdderInput{}
		err := c.BindJSON(body)
		if err != nil {
			c.AbortWithError(500, err)
		}		

		calcTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "calc"}))
		input := &adder.AdderGRPCInput{
			One: body.First,
			Two: body.Second,
		}
		gRPCResult, err := gRPCClient.Add(ctx, input)
		if err != nil {
			c.AbortWithError(500, err)
		}
		calcTimer.ObserveDuration()
		
		result := &adder.AdderResponse{
			Result: gRPCResult.Result,
		}
		c.JSON(200, result)
		totalTimer.ObserveDuration()
	})
	// This endpoint uses a gRPC call over Unix sockets to connect to the sidecar.
	r.POST("/adder-with-sidecar-grpc-socket", func(c *gin.Context) {
		routeName := "adder-with-grpc-sidecar-socket"
		totalTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "total"}))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		body := &adder.AdderInput{}
		err := c.BindJSON(body)
		if err != nil {
			c.AbortWithError(500, err)
		}		

		calcTimer := prometheus.NewTimer(serverResponseTime.With(prometheus.Labels{"route": routeName, "stage": "calc"}))
		input := &adder.AdderGRPCInput{
			One: body.First,
			Two: body.Second,
		}
		gRPCResult, err := gRPCClientSocket.Add(ctx, input)
		if err != nil {
			c.AbortWithError(500, err)
		}
		calcTimer.ObserveDuration()
		
		result := &adder.AdderResponse{
			Result: gRPCResult.Result,
		}
		c.JSON(200, result)
		totalTimer.ObserveDuration()
	})

	adder := &adder.Adder{
		ServerResponsePromClient: serverResponseTime,
	}
	// This endpoint uses the local method.
	r.POST("/adder", adder.ServeHTTP)
	go r.Run()
	<- stopChan
}