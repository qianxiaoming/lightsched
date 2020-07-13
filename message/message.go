package message

import "github.com/qianxiaoming/lightsched/model"

const (
	// KindScheduleTask 调度任务执行消息
	KindScheduleTask = "task"
)

// JSON 表示内容是JSON的消息，其中通过kind字段说明类型
type JSON struct {
	Kind    string
	Content []byte
}

// RegisterNode 节点注册消息
type RegisterNode struct {
	Name      string             `json:"name"`
	Platform  model.PlatformInfo `json:"platform"`
	Labels    map[string]string  `json:"labels,omitempty"`
	Resources model.ResourceSet  `json:"resources"`
}

// TaskStatus 表示任务的执行状态信息
type TaskStatus struct {
	ID       string          `json:"id"`
	State    model.TaskState `json:"state"`
	Progress int             `json:"progress"`
	ExitCode int             `json:"exit_code"`
	Error    string          `json:"error,omitempty"`
}

// Heartbeat 节点心跳信息
type Heartbeat struct {
	Name    string        `json:"name"`
	CPU     float64       `json:"cpu"`
	Memory  float64       `json:"memory"`
	Payload []*TaskStatus `json:"payload,omitempty"`
}
