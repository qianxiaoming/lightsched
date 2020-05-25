package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

// Config 是API Server的配置信息
type Config struct {
	address  string
	port     int
	rpcPort  int
	dataPath string
	logPath  string
}

// APIServer 是集群的中心服务，实现了资源管理、任务调度和API响应等功能
type APIServer struct {
	config        Config
	restRouter    *gin.Engine
	restEndpoints map[string]RestEndpoint
	restChan      chan interface{}
	nodeChan      chan interface{}
	taskChan      chan interface{}
	stopChan      chan struct{}
}

// NewAPIServer 用以创建和初始化API Server实例
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
		restChan:      make(chan interface{}),
		nodeChan:      make(chan interface{}),
		taskChan:      make(chan interface{}),
		stopChan:      make(chan struct{}),
	}
}

// Run 是API Server的主运行逻辑，返回时服务即结束运行
func (svc *APIServer) Run() int {
	fmt.Println("Light Scheduler API Server is starting up...")

	var wg sync.WaitGroup
	waitForStop := func(wait func()) {
		wg.Add(1)
		defer wg.Done()
		wait()
	}
	// 启动API Server的主事件循环
	go waitForStop(func() {
		svc.EventLoop()
	})

	// 启动对内gRPC服务
	grpcAddr := fmt.Sprintf("%s:%d", svc.config.address, svc.config.rpcPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen rpc port %d: %v", svc.config.rpcPort, err)
	}
	grpcsrv := grpc.NewServer()
	//pb.RegisterGreeterServer(s, &server{})
	go waitForStop(func() {
		log.Printf("Start gRPC Service on %v\n", grpcAddr)
		if err := grpcsrv.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC serve: %v", err)
		}
	})

	// 启动对外的RESTful API服务
	gin.SetMode(gin.ReleaseMode)
	svc.registerRestEndpoint(gin.Default())
	httpsrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", svc.config.address, svc.config.port),
		Handler: svc.restRouter,
	}
	go waitForStop(func() {
		log.Printf("Start RESTful API Service on %v\n", httpsrv.Addr)
		if err := httpsrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.address, svc.config.port, err)
		}
	})

	// 等待系统中断信号并关闭gRPC和HTTP服务
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down api server...")

	if err := httpsrv.Shutdown(context.Background()); err != nil {
		log.Fatal("Server Shutdown failed:", err)
	}
	grpcsrv.GracefulStop()
	close(svc.stopChan)
	wg.Wait()

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
