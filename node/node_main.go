package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// Config 是Node Server的配置信息
type Config struct {
	apiserver string
	hostname  string
	heartbeat time.Duration
	logPath   string
}

type Heartbeat struct {
	errors int
	url    string
}

// NodeServer 是集群的工作节点服务。每个执行任务的节点上部署1个NodeServer。
type NodeServer struct {
	config    Config
	resources model.ResourceSet
	platform  model.PlatformInfo
	labels    map[string]string
	state     model.NodeState
	heartbeat Heartbeat
	update    chan interface{}
}

// NewNodeServer 创建1个NodeServer实例
func NewNodeServer(apiserver string, hostname string) *NodeServer {
	if len(apiserver) == 0 {
		apiserver = os.Getenv("LIGHTSCHED_APISERVER_ADDR")
	}
	if len(hostname) == 0 {
		var err error
		if hostname, err = os.Hostname(); err != nil {
			log.Printf("Cannot get the host name of this machine: %v", err)
			return nil
		}
	}
	logPath, _ := filepath.Abs("log")
	return &NodeServer{
		config: Config{
			apiserver: apiserver,
			hostname:  hostname,
			heartbeat: time.Second * 2,
			logPath:   logPath,
		},
		state: model.NodeUnknown,
		heartbeat: Heartbeat{
			errors: 0,
			url:    "http://" + apiserver + "/heartbeat",
		},
	}
}

// Run 是Node Server的主运行逻辑，返回时服务即结束运行
func (node *NodeServer) Run(cpustr string, gpustr string, memorystr string, labelstr string) int {
	log.Println("Light Scheduler Node Server is starting up...")
	log.Printf("    API Server:    %s", node.config.apiserver)
	log.Printf("    Host Name:     %s", node.config.hostname)
	log.Printf("    Log Path:      %s", node.config.logPath)
	log.Printf("    Heartbeat:     %s", node.config.heartbeat)

	// 记录传入的label信息
	if len(labelstr) > 0 {
		node.labels = make(map[string]string)
		labels := strings.Split(labelstr, ";")
		for _, label := range labels {
			kv := strings.Split(label, "=")
			if len(kv) == 2 {
				node.labels[kv[0]] = kv[1]
			}
		}
	}

	// 确定系统的资源信息
	if err := node.collectSystemResources(cpustr, gpustr, memorystr); err != nil {
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
			} else if node.state == model.NodeOnline {
				// 正常状态下发送心跳信息
				if err := node.sendHeartbeat(); err == nil {
					node.heartbeat.errors = 0
				} else {
					node.heartbeat.errors = node.heartbeat.errors + 1
					if node.heartbeat.errors >= 5 {
						node.state = model.NodeUnknown
						node.heartbeat.errors = 0
					} else {
						timeout = node.config.heartbeat * 3
					}
				}
			}
			timer.Reset(timeout)
		}
	}
	log.Println("Server exited")
	return 0
}

func (node *NodeServer) collectSystemResources(cpustr string, gpustr string, memorystr string) error {
	// 获取操作系统平台信息
	if platform, family, version, err := host.PlatformInformation(); err == nil {
		node.platform.Name = platform
		node.platform.Family = family
		node.platform.Version = strings.Split(version, " ")[0]
		if strings.Contains(node.platform.Name, "Windows") {
			node.platform.Kind = constant.PlatformWindows
		} else {
			node.platform.Kind = constant.PlatformLinux
		}
		log.Printf("    Platform:      %s(%s) %s", platform, family, node.platform.Version)
	} else {
		return fmt.Errorf("Unable to get Operation System Information: %v", err)
	}
	// 获取CPU相关信息
	if len(cpustr) == 0 {
		if cc, err := cpu.Counts(true); err == nil {
			node.resources.CPU.Cores = float32(cc)
		} else {
			return fmt.Errorf("Unable to get CPU Cores: %v", err)
		}
		if infos, err := cpu.Info(); err == nil {
			node.resources.CPU.MinFreq = int(infos[0].Mhz)
			node.resources.CPU.Frequency = int(node.resources.CPU.Cores) * node.resources.CPU.MinFreq
		} else {
			return fmt.Errorf("Unable to get CPU Information: %v", err)
		}
	} else {
		cpus := strings.Split(cpustr, ";")
		for _, s := range cpus {
			kv := strings.Split(s, "=")
			val, _ := strconv.Atoi(kv[1])
			if kv[0] == "cores" {
				node.resources.CPU.Cores = float32(val)
			} else if kv[0] == "freq" {
				node.resources.CPU.MinFreq = val
			}
		}
		node.resources.CPU.Frequency = int(node.resources.CPU.Cores) * node.resources.CPU.MinFreq
	}
	log.Printf("    CPU Cores:     %d\n", int(node.resources.CPU.Cores))
	log.Printf("    CPU Frequency: %d\n", node.resources.CPU.MinFreq)
	// 获取内存相关信息
	if len(memorystr) == 0 {
		if v, err := mem.VirtualMemory(); err == nil {
			node.resources.Memory = int(float64(v.Total/1024)/float64(1024)/float64(1024)+0.5) * 1024
		} else {
			return fmt.Errorf("Unable to get system memory: %v", err)
		}
	} else {
		val, _ := strconv.Atoi(memorystr)
		node.resources.Memory = val
	}
	log.Printf("    Total Memory:  %v Mi\n", node.resources.Memory)
	// 获取GPU相关信息
	if len(gpustr) == 0 {
		var smiName string
		if node.platform.Kind == constant.PlatformWindows {
			smiName = "nvidia-smi.exe"
		} else {
			smiName = "nvidia-smi"
		}
		if smi, err := exec.LookPath(smiName); err == nil {
			cmd := exec.Command(smi, "-q", "-d", "MEMORY")
			if output, err := cmd.CombinedOutput(); err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					line = strings.Trim(line, " \r\n")
					if len(line) == 0 {
						continue
					}
					pos := strings.LastIndex(line, ": ")
					if strings.HasPrefix(line, "CUDA") {
						val, _ := strconv.ParseFloat(line[pos+2:], 32)
						node.resources.GPU.CUDA = int(val * 100.0)
					} else if strings.HasPrefix(line, "Attached GPUs") {
						val, _ := strconv.Atoi(line[pos+2:])
						node.resources.GPU.Cards = val
					} else if strings.HasPrefix(line, "Total") {
						pos2 := strings.LastIndex(line, " ")
						val, _ := strconv.Atoi(line[pos+2 : pos2])
						node.resources.GPU.Memory = val / 1024
						break
					}
				}
			}
		} else {
			log.Println("nvidia-smi.exe cannot be found in current path or PATH environment")
			node.resources.GPU.Cards = constant.DefaultNodeGPUCount
			node.resources.GPU.Memory = constant.DefaultNodeGPUMemory
			node.resources.GPU.CUDA = constant.DefaultNodeCUDA
		}
	} else {
		gpus := strings.Split(gpustr, ";")
		for _, s := range gpus {
			kv := strings.Split(s, "=")
			val, _ := strconv.Atoi(kv[1])
			switch kv[0] {
			case "cards":
				node.resources.GPU.Cards = val
			case "mem":
				node.resources.GPU.Memory = val
			case "cuda":
				node.resources.GPU.CUDA = val
			}
		}
	}
	log.Printf("    GPU Cards:     %d\n", node.resources.GPU.Cards)
	log.Printf("    GPU Memory:    %d\n", node.resources.GPU.Memory)
	log.Printf("    CUDA Version:  %d\n", node.resources.GPU.CUDA)
	log.Println("All resources information collected")
	return nil
}

func (node *NodeServer) registerSelf() error {
	if node.state == model.NodeOffline {
		return nil
	}
	log.Printf("Register node to API Server %s as %s...\n", node.config.apiserver, node.config.hostname)
	msg := &message.RegisterNode{
		Name:      node.config.hostname,
		Platform:  node.platform,
		Labels:    node.labels,
		Resources: node.resources,
	}
	content, _ := json.Marshal(msg)
	if resp, err := http.Post("http://"+node.config.apiserver+"/nodes", "application/json", bytes.NewReader(content)); err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			log.Printf("Node registered: %s\n", string(body))
			node.state = model.NodeOnline
			node.heartbeat.errors = 0
		} else {
			log.Printf("Failed to register node to API Server: %s", string(body))
		}
		return nil
	} else {
		log.Printf("Failed to register node to API Server: %v", err)
		return err
	}
}
