package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Config 是API Server的配置信息
type Config struct {
	address  string
	restPort int
	nodePort int
	dataPath string
	logPath  string
}

// HTTPEndpoint 是对不同资源对象提供HTTP API实现的接口
type HTTPEndpoint interface {
	registerRoute()
	restPrefix() string
}

// APIServer 是集群的中心服务，实现了资源管理、任务调度和API响应等功能
type APIServer struct {
	sync.RWMutex

	config        Config
	state         *StateModel
	restRouter    *gin.Engine
	nodeRouter    *gin.Engine
	restEndpoints map[string]HTTPEndpoint
	nodeEndpoints map[string]HTTPEndpoint
}

// NewAPIServer 用以创建和初始化API Server实例
func NewAPIServer() *APIServer {
	return &APIServer{
		config: Config{
			address:  "",
			restPort: 20516,
			nodePort: 20517,
			dataPath: "./data",
			logPath:  "./log",
		},
		state:         NewStateModel(),
		restEndpoints: make(map[string]HTTPEndpoint),
		nodeEndpoints: make(map[string]HTTPEndpoint),
	}
}

// Run 是API Server的主运行逻辑，返回时服务即结束运行
func (svc *APIServer) Run() int {
	log.Println("Light Scheduler API Server is starting up...")
	if err := svc.state.initState(svc.config.dataPath); err != nil {
		log.Printf("Failed to initialize state data: %v\n", err)
		return 1
	}
	defer svc.state.clearState()

	var wg sync.WaitGroup
	waitForStop := func(wait func()) {
		wg.Add(1)
		defer wg.Done()
		wait()
	}

	// gin.SetMode(gin.ReleaseMode)
	// 启动对内节点的HTTP服务
	nodeEngine := gin.New()
	nodeEngine.Use(gin.Recovery())
	svc.registerNodeEndpoint(nodeEngine)
	httpNode := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", svc.config.address, svc.config.nodePort),
		Handler:      nodeEngine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go waitForStop(func() {
		log.Printf("Start Node HTTP Service on %v\n", httpNode.Addr)
		if err := httpNode.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.address, svc.config.nodePort, err)
		}
	})

	// 启动对外的RESTful API服务
	restEngine := gin.New()
	restEngine.Use(gin.Recovery())
	svc.registerRestEndpoint(restEngine)
	httpRest := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", svc.config.address, svc.config.restPort),
		Handler:      restEngine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go waitForStop(func() {
		log.Printf("Start RESTful API Service on %v\n", httpRest.Addr)
		if err := httpRest.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.address, svc.config.restPort, err)
		}
	})

	// 启动定时器并等待系统中断信号
	timer := time.NewTimer(time.Second)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	stopped := false
	for !stopped {
		select {
		case <-quit:
			stopped = true
		case <-timer.C:
			timer.Reset(time.Second)
		}
	}

	// 关闭HTTP服务
	log.Println("Shutting down api server...")
	if err := httpRest.Shutdown(context.Background()); err != nil {
		log.Fatal("Server Shutdown failed:", err)
	}
	if err := httpNode.Shutdown(context.Background()); err != nil {
		log.Fatal("Server Shutdown failed:", err)
	}
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
	registerEndpoint := func(endpoint HTTPEndpoint) {
		svc.restEndpoints[endpoint.restPrefix()] = endpoint
		endpoint.registerRoute()
	}
	// 绑定/jobs相关路径处理
	registerEndpoint(&JobEndpoint{server: svc})
	// 绑定/tasks相关路径处理
	registerEndpoint(&TaskEndpoint{server: svc})
	// 绑定/queues相关路径处理
	registerEndpoint(&QueueEndpoint{server: svc})
	// 绑定/nodes相关路径处理
	registerEndpoint(&NodeEndpoint{server: svc})
}

func (svc *APIServer) registerNodeEndpoint(router *gin.Engine) {
	svc.nodeRouter = router

	// 绑定针对各种资源的RESTful API路径
	registerEndpoint := func(endpoint HTTPEndpoint) {
		svc.nodeEndpoints[endpoint.restPrefix()] = endpoint
		endpoint.registerRoute()
	}
	// 绑定/heartbeat相关路径处理
	registerEndpoint(&HeartbeatEndpoint{server: svc})
}
