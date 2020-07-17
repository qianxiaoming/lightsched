package message

import (
	"strings"
	"time"

	"github.com/qianxiaoming/lightsched/model"
)

const (
	// KindScheduleTask 调度任务执行消息
	KindScheduleTask = "ScheduleTask"
	// KindTerminateJob 终止Job消息
	KindTerminateJob = "TerminateJob"
)

// JSON 表示内容是JSON的消息，其中通过kind字段说明类型
type JSON struct {
	Kind    string `json:"kind"`
	Object  string `json:"object"` // 消息相关的对象标识
	Content []byte `json:"content,omitempty"`
}

// Filter 判断消息是否可以共同发送
func Filter(msg *JSON, other *JSON) bool {
	if msg.Kind == KindTerminateJob && other.Kind == KindScheduleTask {
		return !strings.HasPrefix(other.Object, msg.Object)
	}
	return true
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

// JobInfo 返回给客户端的Job信息
type JobInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Queue       string            `json:"queue"`
	Priority    int               `json:"priority"`
	Labels      map[string]string `json:"labels,omitempty"`
	Taints      map[string]string `json:"taints,omitempty"`
	Schedulable bool              `json:"schedulable"`
	MaxErrors   int               `json:"max_errors"`
	Groups      []string          `json:"groups"`
	SubmitTime  time.Time         `json:"submit_time"`
	ExecTime    *time.Time        `json:"exec_time,omitempty"`
	FinishTime  *time.Time        `json:"finish_time,omitempty"`
	State       model.JobState    `json:"state"`
	Progress    int               `json:"progress"`
	TotalTasks  int               `json:"tasks"`
}

// NewJobInfo 根据Job创建对应的信息体
func NewJobInfo(job *model.Job) *JobInfo {
	info := &JobInfo{
		ID:          job.ID,
		Name:        job.Name,
		Queue:       job.Queue,
		Priority:    job.Priority,
		Labels:      job.Labels,
		Taints:      job.Taints,
		Schedulable: job.Schedulable,
		MaxErrors:   job.MaxErrors,
		Groups:      make([]string, 0, len(job.Groups)),
		SubmitTime:  job.SubmitTime,
		ExecTime:    nil,
		FinishTime:  nil,
		State:       job.State,
		Progress:    job.Progress,
		TotalTasks:  job.TotalTasks,
	}
	for _, g := range job.Groups {
		info.Groups = append(info.Groups, g.Name)
	}
	if !job.ExecTime.IsZero() {
		info.ExecTime = &time.Time{}
		*info.ExecTime = job.ExecTime
	}
	if !job.FinishTime.IsZero() {
		info.FinishTime = &time.Time{}
		*info.FinishTime = job.FinishTime
	}
	return info
}

// NodeInfo 返回给客户端的计算节点信息
type NodeInfo model.WorkNode
