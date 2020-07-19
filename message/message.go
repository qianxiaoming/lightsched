package message

import (
	"strings"
	"time"

	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
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

// TaskReport 是由节点上报的任务执行状态
type TaskReport struct {
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
	Payload []*TaskReport `json:"payload,omitempty"`
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
		ID:         job.ID,
		Name:       job.Name,
		Queue:      job.Queue,
		Priority:   job.Priority,
		Labels:     job.Labels,
		Taints:     job.Taints,
		MaxErrors:  job.MaxErrors,
		Groups:     make([]string, 0, len(job.Groups)),
		SubmitTime: job.SubmitTime,
		ExecTime:   nil,
		FinishTime: nil,
		State:      job.State,
		Progress:   job.Progress,
		TotalTasks: job.TotalTasks,
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

// TaskInfo 返回给客户端的计算任务信息
type TaskInfo struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Envs       []string           `json:"envs,omitempty"`
	Command    string             `json:"command,omitempty"`
	Args       string             `json:"args,omitempty"`
	WorkDir    string             `json:"workdir,omitempty"`
	Labels     map[string]string  `json:"labels,omitempty"`
	Taints     map[string]string  `json:"taints,omitempty"`
	Resources  *model.ResourceSet `json:"resources,omitempty"`
	State      model.TaskState    `json:"state"`
	NodeName   string             `json:"node,omitempty"`
	Progress   int                `json:"progress"`
	ExitCode   int                `json:"exit_code"`
	Error      string             `json:"error,omitempty"`
	StartTime  *time.Time         `json:"start_time,omitempty"`
	FinishTime *time.Time         `json:"finish_time,omitempty"`
}

// NewTaskInfo 根据Task创建对应的信息体
func NewTaskInfo(task *model.Task) *TaskInfo {
	info := &TaskInfo{
		ID:         task.ID,
		Name:       task.Name,
		Envs:       task.Envs,
		Command:    task.Command,
		Args:       task.Args,
		WorkDir:    task.WorkDir,
		Labels:     util.CloneMap(task.Labels),
		Taints:     util.CloneMap(task.Taints),
		Resources:  task.Resources.Clone(),
		State:      task.State,
		NodeName:   task.NodeName,
		Progress:   task.Progress,
		ExitCode:   task.ExitCode,
		Error:      task.Error,
		StartTime:  nil,
		FinishTime: nil,
	}
	if !task.StartTime.IsZero() {
		info.StartTime = &time.Time{}
		*info.StartTime = task.StartTime
	}
	if !task.FinishTime.IsZero() {
		info.FinishTime = &time.Time{}
		*info.FinishTime = task.FinishTime
	}
	return info
}

// TaskStatus 包含任务的执行状态信息
type TaskStatus struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Labels     map[string]string `json:"labels,omitempty"`
	State      model.TaskState   `json:"state"`
	NodeName   string            `json:"node,omitempty"`
	Progress   int               `json:"progress"`
	ExitCode   int               `json:"exit_code"`
	Error      string            `json:"error,omitempty"`
	StartTime  *time.Time        `json:"start_time,omitempty"`
	FinishTime *time.Time        `json:"finish_time,omitempty"`
}

// NewTaskStatus 根据Task创建对应的状态信息体
func NewTaskStatus(task *model.Task) *TaskStatus {
	info := &TaskStatus{
		ID:         task.ID,
		Name:       task.Name,
		Labels:     task.Labels,
		State:      task.State,
		NodeName:   task.NodeName,
		Progress:   task.Progress,
		ExitCode:   task.ExitCode,
		Error:      task.Error,
		StartTime:  nil,
		FinishTime: nil,
	}
	if !task.StartTime.IsZero() {
		info.StartTime = &time.Time{}
		*info.StartTime = task.StartTime
	}
	if !task.FinishTime.IsZero() {
		info.FinishTime = &time.Time{}
		*info.FinishTime = task.FinishTime
	}
	return info
}

// JobQueueInfo 返回给客户端的计算作业队列信息
type JobQueueInfo model.JobQueue
