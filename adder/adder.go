package adder

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	UnixSocket string = "/var/run/adder"
)

type Adder struct {
	ServerResponsePromClient *prometheus.HistogramVec
	UnimplementedAdderServer
}
type AdderInput struct {
	First float64 `json:"first"`
	Second float64 `json:"second"`
}
type AdderResponse struct {
	Result float64 `json:"response"`
}

func (a *Adder) add(first float64, second float64) float64 {
	return first + second
}

// Add wraps the existing `add` method for use with the gRPC implementation
func (a *Adder) Add(ctx context.Context, input *AdderGRPCInput) (*AdderGRPCResult, error) {
	first := input.One
	second := input.Two
	result := &AdderGRPCResult{
		Result: a.add(first, second),
	}
	return result, nil
}

func (a *Adder) ServeHTTP(ctx *gin.Context) {
	routeName := "adder"
	var timer *prometheus.Timer
	if a.ServerResponsePromClient != nil {
		timer = prometheus.NewTimer(a.ServerResponsePromClient.With(prometheus.Labels{"route":routeName, "stage":"total"}))
	}
	var input AdderInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.AbortWithError(500, err)
	}

	var calcTimer *prometheus.Timer
	if a.ServerResponsePromClient != nil {
		calcTimer = prometheus.NewTimer(a.ServerResponsePromClient.With(prometheus.Labels{"route": routeName, "stage": "calc"}))
	}
	returnVal := a.add(input.First, input.Second)
	if calcTimer != nil {
		calcTimer.ObserveDuration()
	}
	
	response := &AdderResponse{
		Result: returnVal,
	}
	ctx.JSON(200, response)
	if timer != nil {
		timer.ObserveDuration()
	}
}