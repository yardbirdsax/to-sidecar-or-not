package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"


	"github.com/yardbirdsax/to-sidecar-or-not/adder"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		zap.L().Panic(err.Error())
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	tcpList, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", port))
	if err != nil {
		zap.L().Panic(err.Error())
	}
	defer tcpList.Close()

	sockAddr := os.Getenv("SOCK_ADDR")
	if sockAddr == "" {
		zap.L().Panic("required env variable SOCK_ADDR is not set")
	}
	unixList, err := net.Listen("unix", sockAddr)
	if err != nil {
		zap.L().Panic(err.Error())
	}
	defer unixList.Close()

	opts := []grpc.ServerOption{}
	tcpServ := grpc.NewServer(opts...)
	unixServ := grpc.NewServer(opts...)
	adderServ := &adder.Adder{}
	adder.RegisterAdderServer(tcpServ, adderServ)
	adder.RegisterAdderServer(unixServ, adderServ)
	errChan := make(chan error, 2)
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		zap.L().Info("starting TCP server")
		err := tcpServ.Serve(tcpList)
		if err != nil {
			errChan <- err
			zap.L().Panic(err.Error())	
		}
	}()
	go func() {
		zap.L().Info("starting unix socket server")
		err := unixServ.Serve(unixList)
		if err != nil {
			errChan <- err
			zap.L().Panic(err.Error())
		}
	}()
	var e error
	var sig os.Signal
	select {
	case e = <- errChan:
		zap.L().Panic(e.Error())
	case sig = <- intChan:
		zap.L().Info(fmt.Sprintf("received term signal %s, shutting down", sig))
		unixServ.GracefulStop()
		tcpServ.GracefulStop()
		return
	}
	zap.L().Warn("this shouldn't be logged")
}