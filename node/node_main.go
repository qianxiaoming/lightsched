package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/qianxiaoming/lightsched/util"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// Config 是Node Server的配置信息
type Config struct {
	Apiserver string        `json:"server"`
	Hostname  string        `json:"hostname"`
	Heartbeat time.Duration `json:"-"`
	LogPath   string        `json:"log_path"`
	LogURL    string        `json:"-"`
}

type TaskUpdate struct {
	status  *message.TaskReport
	process *os.Process
}

type Heartbeat struct {
	errors  int
	url     string
	payload map[string]*message.TaskReport
}

type TaskProcess struct {
	process *os.Process
	killed  bool
}

// NodeServer 是集群的工作节点服务。每个执行任务的节点上部署1个NodeServer。
type NodeServer struct {
	config      Config
	resources   model.ResourceSet
	platform    model.PlatformInfo
	labels      map[string]string
	state       model.NodeState
	registering bool
	heartbeat   Heartbeat
	executings  map[string]TaskProcess // 正在运行的Task信息
	update      chan *TaskUpdate
}

// NewNodeServer 创建一个NodeServer实例
func NewNodeServer(confPath string, apiserver string, hostname string, heartbeat int) *NodeServer {
	// 尝试从配置文件加载设置信息
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
			conf.Heartbeat = time.Second * 2
			if len(conf.Hostname) == 0 {
				conf.Hostname, _ = os.Hostname()
			}
		}
	} else {
		log.Println("No configuration file found and default setting will be used")
	}
	if conf == nil {
		logPath, _ := filepath.Abs("log")
		name, _ := os.Hostname()
		conf = &Config{
			Apiserver: os.Getenv("LIGHTSCHED_APISERVER"),
			Hostname:  name,
			Heartbeat: time.Second * 2,
			LogPath:   logPath,
		}
		if len(conf.Apiserver) == 0 {
			conf.Apiserver = fmt.Sprintf("127.0.0.1:%d", constant.DefaultNodePort)
		}
		b, _ := json.MarshalIndent(conf, "", "  ")
		if err := ioutil.WriteFile(constant.NodeSeverConfigFile, b, 0666); err != nil {
			log.Printf("Unable to write config file %s: %v\n", constant.NodeSeverConfigFile, err)
		}
	}
	if len(apiserver) != 0 {
		conf.Apiserver = apiserver
	}
	conf.LogURL = "http://" + conf.Apiserver + "/tasks/%s/log"
	if len(hostname) != 0 {
		conf.Hostname = hostname
	}
	if heartbeat > 0 {
		conf.Heartbeat = time.Second * time.Duration(heartbeat)
	}

	// 配置日志信息
	if len(conf.LogPath) > 0 {
		if err := util.MakeDirAll(conf.LogPath); err != nil {
			log.Printf("Cannot create log directory %s: %v\n", conf.LogPath, err)
		} else {
			filename := filepath.Join(conf.LogPath, fmt.Sprintf(constant.NodeSeverLogFile, conf.Hostname))
			if logFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766); err != nil {
				log.Printf("Cannot open log file %s: %v\n", filename, err)
			} else {
				log.SetOutput(io.MultiWriter(os.Stdout, logFile))
				log.SetFlags(log.LstdFlags | log.LUTC)
			}
		}
	}

	return &NodeServer{
		config:      *conf,
		state:       model.NodeUnknown,
		registering: false,
		heartbeat: Heartbeat{
			errors:  0,
			url:     "http://" + conf.Apiserver + "/heartbeat",
			payload: make(map[string]*message.TaskReport),
		},
		executings: make(map[string]TaskProcess),
		update:     make(chan *TaskUpdate, 32),
	}
}

// Run 是Node Server的主运行逻辑，返回时服务即结束运行
func (node *NodeServer) Run(cpustr string, gpustr string, memorystr string, labelstr string) int {
	log.Println("Light Scheduler Node Server is starting up...")
	log.Printf("    API Server:    %s", node.config.Apiserver)
	log.Printf("    Host Name:     %s", node.config.Hostname)
	log.Printf("    Log Path:      %s", node.config.LogPath)
	log.Printf("    Heartbeat:     %s", node.config.Heartbeat)

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
	timer := time.NewTimer(node.config.Heartbeat)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	stopped := false
	for !stopped {
		select {
		case <-quit:
			stopped = true
		case update := <-node.update:
			// 更新Task的最新状态。状态信息将在心跳中统一发给服务端。
			if last, ok := node.heartbeat.payload[update.status.ID]; ok && last.State == update.status.State {
				last.Progress = update.status.Progress
				if len(update.status.Error) > 0 {
					if len(last.Error) == 0 {
						last.Error = update.status.Error
					} else {
						last.Error = last.Error + ";" + update.status.Error
					}
				}
			} else {
				if ok && len(last.Error) != 0 {
					if len(update.status.Error) == 0 {
						update.status.Error = last.Error
					} else {
						update.status.Error = last.Error + ";" + update.status.Error
					}
				}
				node.heartbeat.payload[update.status.ID] = update.status
			}
			// 检查是否在“正在运行任务”列表中
			if proc, ok := node.executings[update.status.ID]; ok {
				if update.status.State != model.TaskExecuting {
					if update.status.State == model.TaskFailed && proc.killed {
						node.heartbeat.payload[update.status.ID].State = model.TaskTerminated
					}
					delete(node.executings, update.status.ID)
				}
			} else if update.status.State == model.TaskExecuting && update.process != nil {
				node.executings[update.status.ID] = TaskProcess{update.process, false}
			}
		case <-timer.C:
			timeout := node.config.Heartbeat
			if node.state == model.NodeUnknown {
				// 初始状态或访问API Server失败，进入重新注册阶段
				if err := node.registerSelf(); err != nil {
					timeout = node.config.Heartbeat * 5
				}
			} else if node.state == model.NodeOnline {
				// 正常状态下发送心跳信息
				if err := node.sendHeartbeat(); err == nil {
					node.heartbeat.errors = 0
				} else {
					// 是否需要立刻重新注册
					if err == errNodeNotRegistered {
						node.state = model.NodeUnknown
						node.heartbeat.errors = 0
					} else {
						// 心跳发送失败时增加失败计数。当计数累加到5时进入未注册状态。
						node.heartbeat.errors = node.heartbeat.errors + 1
						if node.heartbeat.errors > 3 {
							node.state = model.NodeUnknown
							node.heartbeat.errors = 0
						} else {
							timeout = node.config.Heartbeat * 3
						}
					}
				}
			}
			timer.Reset(timeout)
		}
	}
	log.Println("Server exited")
	return 0
}

func (node *NodeServer) notifyTaskStatus(id string, state model.TaskState, process *os.Process, progress, exit int, err string) {
	node.update <- &TaskUpdate{
		status: &message.TaskReport{
			ID:       id,
			State:    state,
			Progress: progress,
			ExitCode: exit,
			Error:    err,
		},
		process: process,
	}
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
	if node.registering == false {
		log.Printf("Register node to API Server %s as %s...\n", node.config.Apiserver, node.config.Hostname)
		node.registering = true
	}
	msg := &message.RegisterNode{
		Name:      node.config.Hostname,
		Platform:  node.platform,
		Labels:    node.labels,
		Resources: node.resources,
	}
	content, _ := json.Marshal(msg)
	if resp, err := http.Post("http://"+node.config.Apiserver+"/nodes", "application/json", bytes.NewReader(content)); err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusOK {
			log.Printf("Node registered: %s\n", string(body))
			node.state = model.NodeOnline
			node.heartbeat.errors = 0
			node.registering = false
		} else {
			log.Printf("Failed to register node to API Server: %s", string(body))
		}
		return nil
	} else {
		//log.Printf("Failed to register node to API Server: %v", err)
		return err
	}
}
