package adder

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type Adder struct {
	ServerResponsePromClient *prometheus.HistogramVec
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

func (a *Adder) ServeHTTP(ctx *gin.Context) {
	var timer *prometheus.Timer
	if a.ServerResponsePromClient != nil {
		timer = prometheus.NewTimer(a.ServerResponsePromClient.With(prometheus.Labels{"route":"addr", "stage":"total"}))
	}
	var input AdderInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.AbortWithError(500, err)
	}
	returnVal := a.add(input.First, input.Second)
	response := &AdderResponse{
		Result: returnVal,
	}
	ctx.JSON(200, response)
	if timer != nil {
		timer.ObserveDuration()
	}
}