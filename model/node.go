package model

import "time"

// NodeState 表示计算节点的状态
type NodeState int32

const (
	// NodeOnline 表示计算节点在线，可分配任务
	NodeOnline NodeState = iota
	// NodeOffline 表示计算节点离线，不可分配任务，但是可以随时切换为在线
	NodeOffline
	// NodeUnknown 表示计算节点状态未知，不可访问
	NodeUnknown
)

// WorkNode 代表实际执行计算任务的节点
type WorkNode struct {
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Port      int               `json:"port"`
	OS        string            `json:"os"`
	State     NodeState         `json:"state"`
	Online    time.Time         `json:"online"`
	Labels    map[string]string `json:"labels,omitempty"`
	Taints    map[string]string `json:"taints,omitempty"`
	Resources ResourceSet       `json:"resources"`
	Reserved  ResourceSet       `json:"reserved"`
}

// NewWorkNode 创建计算节点对象。计算节点默认保留1.5个CPU和2Gi内存。
func NewWorkNode(name string) *WorkNode {
	return &WorkNode{
		Name:  name,
		State: NodeUnknown,
		Reserved: ResourceSet{
			CPU: ResourceCPU{
				Cores: 1.5,
			},
			Memory: 2000,
		},
	}
}
