package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	address  string
	port     int
	rpcPort  int
	dataPath string
	logPath  string
}

type APIServer struct {
	config Config
}

func NewAPIServer() *APIServer {
	return &APIServer{
		config: Config{
			address:  "",
			port:     20516,
			rpcPort:  20517,
			dataPath: "./data",
			logPath:  "./log",
		},
	}
}

func (svc *APIServer) Run() int {
	fmt.Println("Light Scheduler API Server is starting up...")
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	svc.buildHTTPRoute(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", svc.config.address, svc.config.port),
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.address, svc.config.port, err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 3 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shut down api server in 3 seconds...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown failed:", err)
	}
	log.Println("Server exited")
	return 0
}

func (svc *APIServer) buildHTTPRoute(router *gin.Engine) {
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
}
