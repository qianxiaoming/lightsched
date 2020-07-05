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
	Name      string
	Address   string
	Port      int
	OS        string
	State     NodeState
	Online    time.Time
	Labels    map[string]string
	resources ResourceSet
}
