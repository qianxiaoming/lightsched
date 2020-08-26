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

// PlatformInfo 包含操作系统相关的信息
type PlatformInfo struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Family  string `json:"family"`
	Version string `json:"version"`
}

// WorkNode 代表实际执行计算任务的节点
type WorkNode struct {
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Platform  PlatformInfo      `json:"platform"`
	State     NodeState         `json:"state"`
	Online    time.Time         `json:"online"`
	Labels    map[string]string `json:"labels,omitempty"`
	Taints    map[string]string `json:"taints,omitempty"`
	Resources *ResourceSet      `json:"resources"` // 节点的总资源量
	Reserved  *ResourceSet      `json:"reserved"`  // 节点保留的资源量（不用于计算任务调度）
	Available *ResourceSet      `json:"available"` // 在节点刚加入的时候 Available = Resources - Reserved
}

// NewWorkNode 创建计算节点对象。计算节点默认保留2个CPU和4Gi内存。
func NewWorkNode(name string) *WorkNode {
	return &WorkNode{
		Name:      name,
		State:     NodeUnknown,
		Resources: &ResourceSet{},
		Reserved: &ResourceSet{
			CPU: ResourceCPU{
				Cores: 2,
			},
			Memory: 4096,
		},
		Available: &ResourceSet{},
	}
}
