package constant

const (
	// DefaultRestPort 是默认的对外RESTful API服务端口
	DefaultRestPort = 20516
	// DefaultNodePort 是默认的对内部节点服务的端口
	DefaultNodePort = 20517
	// DefaultNodeGPUCount 是在无法正确获取GPU信息时节点默认的GPU卡数
	DefaultNodeGPUCount = 1
	// DefaultNodeGPUMemory 是在无法正确获取GPU信息时节点默认的GPU显存
	DefaultNodeGPUMemory = 12
	// DefaultNodeCUDA 是在无法正确获取GPU信息时节点默认的CUDA版本
	DefaultNodeCUDA = 1020
)

const (
	// APISeverConfigFile 是API Server的配置文件名称
	APISeverConfigFile = "api-server.conf"
	// APISeverLogFile 是API Server的日志文件名称
	APISeverLogFile = "api-server.log"
	// NodeSeverConfigFile 是Node Server的配置文件名称
	NodeSeverConfigFile = "node-server.conf"
	// NodeSeverLogFile 是Node Server的日志文件名称
	NodeSeverLogFile = "node-%s.log"
	// DatabaseFileName 是系统默认主数据库的名字
	DatabaseFileName = "lightsched.db"
	// DefaultQueueName 是默认作业队列的名字
	DefaultQueueName = "default"
	// PlatformWindows 是Windows平台的代号
	PlatformWindows = "Windows"
	// PlatformLinux 是Linux平台的代号
	PlatformLinux = "Linux"
)
