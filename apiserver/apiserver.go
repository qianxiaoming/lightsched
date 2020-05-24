package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	config        Config
	restRouter    *gin.Engine
	restEndpoints map[string]RestEndpoint
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
		restEndpoints: make(map[string]RestEndpoint),
	}
}

func (svc *APIServer) Run() int {
	fmt.Println("Light Scheduler API Server is starting up...")

	// 启动API Server的主事件循环
	go func() {
		svc.EventLoop()
	}()

	//gin.SetMode(gin.ReleaseMode)
	svc.registerRestEndpoint(gin.Default())

	// 启动对外的RESTful API服务
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", svc.config.address, svc.config.port),
		Handler: svc.restRouter,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.address, svc.config.port, err)
		}
	}()

	// 等待系统中断信号并在3秒后关闭HTTP服务
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shut down api server in 3 seconds...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown failed:", err)
	}
	select {
	case <-ctx.Done():
		log.Println("timeout of 3 seconds.")
	}
	log.Println("Server exited")
	return 0
}

func (svc *APIServer) registerRestEndpoint(router *gin.Engine) {
	svc.restRouter = router
	// 绑定系统级API路径实现
	svc.restRouter.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// 绑定针对各种资源的RESTful API路径
	registerEndpoint := func(endpoint RestEndpoint) {
		svc.restEndpoints[endpoint.restPrefix()] = endpoint
		endpoint.registerRoute()
	}
	// 绑定/jobs相关路径处理
	registerEndpoint(&JobEndpoint{handler: svc})
	// 绑定/tasks相关路径处理
	registerEndpoint(&TaskEndpoint{handler: svc})
	// 绑定/queues相关路径处理
	registerEndpoint(&QueueEndpoint{handler: svc})
	// 绑定/nodes相关路径处理
	registerEndpoint(&NodeEndpoint{handler: svc})
	// 绑定/users相关路径处理
	registerEndpoint(&UserEndpoint{handler: svc})
}
