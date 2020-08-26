package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/data"
	"github.com/qianxiaoming/lightsched/util"
)

// Config 是API Server的配置信息
type Config struct {
	Cluster  string `json:"cluster"`
	Address  string `json:"address"`
	RestPort int    `json:"rest"`
	NodePort int    `json:"node"`
	Offline  int    `json:"offline"`
	SchedLog bool   `json:"sched_log"`
	DataPath string `json:"data_path"`
	LogPath  string `json:"log_path"`
}

// HTTPEndpoint 是对不同资源对象提供HTTP API实现的接口
type HTTPEndpoint interface {
	registerRoute()
	restPrefix() string
}

// APIServer 是集群的中心服务，实现了资源管理、任务调度和API响应等功能
type APIServer struct {
	config        Config
	state         *data.StateStore
	nodes         *data.NodeCache
	schedFlag     int32
	schedCycle    int64
	restRouter    *gin.Engine
	nodeRouter    *gin.Engine
	restEndpoints map[string]HTTPEndpoint
	nodeEndpoints map[string]HTTPEndpoint
}

var apiserver *APIServer

// NewAPIServer 用以创建和初始化API Server实例
func NewAPIServer(confPath string) *APIServer {
	// 尝试从配置文件加载设置信息。如果配置文件不存在，服务启动时将会按照默认值自动生成。
	var conf *Config
	if util.PathExists(confPath) {
		log.Printf("Load configuration from %s\n", confPath)
		if b, err := ioutil.ReadFile(confPath); err != nil {
			log.Printf("Unable to read config file %s: %v\n", confPath, err)
		} else {
			conf = &Config{}
			if err := json.Unmarshal(b, conf); err != nil {
				log.Printf("Illegal format of the config file %s: %v\n", confPath, err)
				conf = nil
			}
		}
	} else {
		log.Println("No configuration file found and default setting will be used")
	}

	// 生成默认配置
	dataPath, _ := filepath.Abs("cluster")
	logPath, _ := filepath.Abs("log")
	apiserver = &APIServer{
		config: Config{
			Cluster:  util.GenerateUUID(),
			Address:  "",
			RestPort: constant.DefaultRestPort,
			NodePort: constant.DefaultNodePort,
			Offline:  32,
			SchedLog: false,
			DataPath: dataPath,
			LogPath:  logPath,
		},
		state:         data.NewStateStore(),
		nodes:         data.NewNodeCache(),
		schedFlag:     0,
		schedCycle:    0,
		restEndpoints: make(map[string]HTTPEndpoint),
		nodeEndpoints: make(map[string]HTTPEndpoint),
	}
	if conf == nil {
		b, _ := json.MarshalIndent(apiserver.config, "", "  ")
		if err := ioutil.WriteFile(constant.APISeverConfigFile, b, 0666); err != nil {
			log.Printf("Unable to write config file %s: %v\n", constant.APISeverConfigFile, err)
		}
	} else {
		if len(conf.Cluster) != 0 {
			apiserver.config.Cluster = conf.Cluster
		}
		if len(conf.Address) != 0 {
			apiserver.config.Address = conf.Address
		}
		if conf.RestPort != 0 {
			apiserver.config.RestPort = conf.RestPort
		}
		if conf.NodePort != 0 {
			apiserver.config.NodePort = conf.NodePort
		}
		if conf.Offline != 0 {
			apiserver.config.Offline = conf.Offline
		}
		apiserver.config.SchedLog = conf.SchedLog
		if len(conf.DataPath) != 0 {
			apiserver.config.DataPath = conf.DataPath
		}
		if len(conf.LogPath) != 0 {
			apiserver.config.LogPath = conf.LogPath
		}
	}

	// 配置日志信息
	if len(apiserver.config.LogPath) > 0 {
		if err := util.MakeDirAll(apiserver.config.LogPath); err != nil {
			log.Printf("Cannot create log directory %s: %v\n", apiserver.config.LogPath, err)
		} else {
			filename := filepath.Join(apiserver.config.LogPath, constant.APISeverLogFile)
			if logFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766); err != nil {
				log.Printf("Cannot open log file %s: %v\n", filename, err)
			} else {
				log.SetOutput(io.MultiWriter(os.Stdout, logFile))
				log.SetFlags(log.LstdFlags | log.LUTC)
			}
		}
	}
	return apiserver
}

// Run 是API Server的主运行逻辑，返回时服务即结束运行
func (svc *APIServer) Run() int {
	log.Printf("Light Scheduler API Server is starting up with cluster id \"%s\"...\n", svc.config.Cluster)
	if err := svc.state.InitState(svc.config.DataPath); err != nil {
		log.Printf("Failed to initialize state data: %v\n", err)
		return 1
	}
	defer svc.state.ClearState()

	var wg sync.WaitGroup
	gin.SetMode(gin.ReleaseMode)
	// 启动对内节点的HTTP服务
	nodeEngine := gin.New()
	nodeEngine.Use(gin.Recovery())
	svc.registerNodeEndpoint(nodeEngine)
	httpNode := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", svc.config.Address, svc.config.NodePort),
		Handler:      nodeEngine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go util.WaitForStop(&wg, func() {
		log.Printf("Start Node HTTP Service on \"%v\"\n", httpNode.Addr)
		if err := httpNode.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.Address, svc.config.NodePort, err)
		}
	})

	// 启动对外的RESTful API服务
	restEngine := gin.New()
	restEngine.Use(gin.Recovery())
	svc.registerRestEndpoint(restEngine)
	restEngine.Static("/portal", "./html")
	restEngine.StaticFile("/favicon.ico", "./html/favicon.ico")
	httpRest := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", svc.config.Address, svc.config.RestPort),
		Handler:      restEngine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go util.WaitForStop(&wg, func() {
		log.Printf("Start RESTful API Service on \"%v\"\n", httpRest.Addr)
		if err := httpRest.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cannot listen on %s:%d: %s\n", svc.config.Address, svc.config.RestPort, err)
		}
	})

	// 启动定时器并等待系统中断信号
	timerSched := time.NewTimer(time.Second)
	timerNode := time.NewTimer(time.Second * time.Duration(svc.config.Offline+1))
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	stopped := false
	for !stopped {
		select {
		case <-quit:
			stopped = true
		case <-timerSched.C:
			svc.runScheduleCycle()
			timerSched.Reset(time.Second)
		case <-timerNode.C:
			svc.requestCheckNodes()
			timerNode.Reset(time.Second * time.Duration(svc.config.Offline+1))
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

func (svc *APIServer) setScheduleFlag() {
	atomic.AddInt32(&svc.schedFlag, 1)
}

func (svc *APIServer) registerRestEndpoint(router *gin.Engine) {
	svc.restRouter = router
	// 绑定系统级API路径实现
	svc.restRouter.GET("/cluster", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":    svc.config.Cluster,
			"cycle": svc.schedCycle,
		})
	})

	// 绑定针对各种资源的RESTful API路径
	registerEndpoint := func(endpoint HTTPEndpoint) {
		svc.restEndpoints[endpoint.restPrefix()] = endpoint
		endpoint.registerRoute()
	}
	// 绑定/jobs相关路径处理
	registerEndpoint(&JobEndpoint{})
	// 绑定/tasks相关路径处理
	registerEndpoint(&TaskEndpoint{})
	// 绑定/queues相关路径处理
	registerEndpoint(&QueueEndpoint{})
	// 绑定/nodes相关路径处理
	registerEndpoint(&NodeEndpoint{})
}

func (svc *APIServer) registerNodeEndpoint(router *gin.Engine) {
	svc.nodeRouter = router

	// 绑定针对各种资源的RESTful API路径
	registerEndpoint := func(endpoint HTTPEndpoint) {
		svc.nodeEndpoints[endpoint.restPrefix()] = endpoint
		endpoint.registerRoute()
	}
	// 绑定/heartbeat相关路径处理
	registerEndpoint(&HeartbeatEndpoint{})
	registerEndpoint(&NodeRegisterEndpoint{})
	registerEndpoint(&TaskLogEndpoint{})
}
