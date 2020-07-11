package node

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// Config 是Node Server的配置信息
type Config struct {
	apiserver string
	heartbeat time.Duration
	logPath   string
}

// NodeServer 是集群的工作节点服务。每个执行任务的节点上部署1个NodeServer。
type NodeServer struct {
	config    Config
	resources model.ResourceSet
	platform  model.PlatformInfo
	state     model.NodeState
	update    chan interface{}
}

// NewNodeServer 创建1个NodeServer实例
func NewNodeServer(apiserver string) *NodeServer {
	if len(apiserver) == 0 {
		apiserver = os.Getenv("LIGHTSCHED_APISERVER_ADDR")
	}
	logPath, _ := filepath.Abs("log")
	return &NodeServer{
		config: Config{
			apiserver: apiserver,
			heartbeat: time.Second * 2,
			logPath:   logPath,
		},
		state: model.NodeUnknown,
	}
}

// Run 是Node Server的主运行逻辑，返回时服务即结束运行
func (node *NodeServer) Run() int {
	log.Println("Light Scheduler Node Server is starting up...")
	log.Printf("    API Server:         %s", node.config.apiserver)
	log.Printf("    Log Path:           %s", node.config.logPath)
	log.Printf("    Heartbeat Interval: %s", node.config.heartbeat)

	// 确定系统的资源信息
	if err := node.collectSystemResources(); err != nil {
		log.Printf("Failed to collect system resources: %v\n", err)
		return 1
	}

	// 启动定时器并等待系统中断信号
	timer := time.NewTimer(node.config.heartbeat)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	stopped := false
	for !stopped {
		select {
		case <-quit:
			stopped = true
		case <-node.update:

		case <-timer.C:
			timeout := node.config.heartbeat
			if node.state == model.NodeUnknown {
				// 初始状态或访问API Server失败，进入重新注册阶段
				if err := node.registerSelf(); err != nil {
					timeout = node.config.heartbeat * 5
				}
			}
			timer.Reset(timeout)
		}
	}
	log.Println("Server exited")
	return 0
}

func (node *NodeServer) collectSystemResources() error {
	if platform, family, version, err := host.PlatformInformation(); err == nil {
		node.platform.Name = platform
		node.platform.Family = family
		node.platform.Version = strings.Split(version, " ")[0]
		if strings.Contains(node.platform.Name, "Windows") {
			node.platform.Kind = constant.PlatformWindows
		} else {
			node.platform.Kind = constant.PlatformLinux
		}
		log.Printf("    Platform: %s(%s) %s", platform, family, node.platform.Version)
	} else {
		return fmt.Errorf("Unable to get Operation System Information: %v", err)
	}
	if cc, err := cpu.Counts(true); err == nil {
		node.resources.CPU.Cores = float32(cc)
		log.Printf("    CPU Logical Cores: %d\n", cc)
	} else {
		return fmt.Errorf("Unable to get CPU Cores: %v", err)
	}
	if infos, err := cpu.Info(); err == nil {
		node.resources.CPU.MinFreq = int(infos[0].Mhz)
		node.resources.CPU.Frequency = int(node.resources.CPU.Cores) * node.resources.CPU.MinFreq
		log.Printf("    CPU Frequency: %d\n", node.resources.CPU.MinFreq)
	} else {
		return fmt.Errorf("Unable to get CPU Information: %v", err)
	}
	if v, err := mem.VirtualMemory(); err == nil {
		node.resources.Memory = int(float64(v.Total/1024)/float64(1024)/float64(1024)+0.5) * 1024
		log.Printf("    Total Memory: %v(%vMi)\n", v.Total, node.resources.Memory)
	} else {
		return fmt.Errorf("Unable to get system memory: %v", err)
	}
	return nil
}

func (node *NodeServer) registerSelf() error {
	return nil
}
