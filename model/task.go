package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qianxiaoming/lightsched/util"
)

// TaskState 表示Task的状态
type TaskState int32

const (
	// TaskQueued 任务等待调度
	TaskQueued TaskState = iota
	// TaskScheduled 任务已经分配到某个节点
	TaskScheduled
	// TaskDispatching 任务正在被分发到调度的节点
	TaskDispatching
	// TaskExecuting 任务正在执行
	TaskExecuting
	// TaskCompleted 任务已经成功结束
	TaskCompleted
	// TaskFailed 任务执行失败
	TaskFailed
	// TaskAborted 任务发生异常结束
	TaskAborted
	// TaskTerminated 任务被取消
	TaskTerminated
)

// IsFinishState 判断Task的状态是否表示已经运行结束
func IsFinishState(state TaskState) bool {
	return state == TaskCompleted || state == TaskFailed || state == TaskAborted || state == TaskTerminated
}

func TaskStateToString(state TaskState) string {
	switch state {
	case TaskQueued:
		return "Queued"
	case TaskScheduled:
		return "Scheduled"
	case TaskDispatching:
		return "Dispatching"
	case TaskExecuting:
		return "Executing"
	case TaskCompleted:
		return "Completed"
	case TaskFailed:
		return "Failed"
	case TaskAborted:
		return "Aborted"
	case TaskTerminated:
		return "Terminated"
	}
	return ""
}

// TaskSpec 指定任务的执行信息
type TaskSpec struct {
	Name          string            `json:"name"`
	Envs          []string          `json:"envs,omitempty"`
	Command       string            `json:"command,omitempty"`
	Args          string            `json:"args,omitempty"`
	WorkDir       string            `json:"workdir,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	*ResourceSpec `json:"resources,omitempty"`
}

// Task 是具体执行的计算任务
type Task struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Envs       []string          `json:"envs,omitempty"`
	Command    string            `json:"command,omitempty"`
	Args       string            `json:"args,omitempty"`
	WorkDir    string            `json:"workdir,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Taints     map[string]string `json:"taints,omitempty"`
	Resources  *ResourceSet      `json:"resources,omitempty"`
	State      TaskState         `json:"state"`
	NodeName   string            `json:"node,omitempty"`
	Progress   int               `json:"-"`
	ExitCode   int               `json:"exit_code"`
	Error      string            `json:"error"`
	StartTime  time.Time         `json:"start_time"`
	FinishTime time.Time         `json:"finish_time"`
}

// NewTaskWithSpec 根据指定的TaskSpec内容创建对应的Task对象
func NewTaskWithSpec(group *TaskGroup, id int, spec *TaskSpec) *Task {
	task := &Task{
		ID:        fmt.Sprintf("%s.%d", group.ID, id),
		Name:      spec.Name,
		Envs:      spec.Envs,
		Command:   strings.ReplaceAll(spec.Command, "\\", "/"),
		Args:      spec.Args,
		WorkDir:   spec.WorkDir,
		Labels:    spec.Labels,
		Resources: NewResourceSetWithSpec(spec.ResourceSpec),
		State:     TaskQueued,
		Progress:  0,
		ExitCode:  -1,
	}
	// 如果Task没有指定一些信息，则将所属TaskGroup的信息赋予它
	if len(task.Command) == 0 {
		task.Command = group.Command
	}
	if len(task.WorkDir) == 0 {
		task.WorkDir = group.WorkDir
	}
	// 如果Task没有指定所需资源，则使用TaskGroup的资源；若都没有指定，使用预定义的默认资源
	if task.Resources == nil {
		task.Resources = group.Resources
		if task.Resources == nil {
			task.Resources = DefaultResourceSet
		}
	}
	if task.Resources.CPU.Cores == 0.0 && task.Resources.CPU.Frequency == 0 &&
		task.Resources.GPU.Cards == 0 && task.Resources.Memory == 0 {
		// 如果没有指定主要资源需求，则将CPU和内存赋予默认值
		task.Resources.CPU = DefaultResourceSet.CPU
		task.Resources.Memory = DefaultResourceSet.Memory
	}
	// 环境变量和标签采取合并的方式
	task.Envs = util.MergeStringSlice(task.Envs, group.Envs)
	task.Labels = util.MergeStringMap(task.Labels, group.Labels)
	return task
}

// ParseTaskID 解析Task的完整编号，分别返回Job, TaskGroup和Task的编号
func ParseTaskID(id string) (string, int, int) {
	ids := strings.Split(id, ".")
	group, _ := strconv.Atoi(ids[1])
	task, _ := strconv.Atoi(ids[2])
	return ids[0], group, task
}

// TaskGroupSpec 表示指定任务组的执行信息，其中包含多个任务描述。
type TaskGroupSpec struct {
	Name          string            `json:"name"`
	Command       string            `json:"command,omitempty"`
	WorkDir       string            `json:"workdir,omitempty"`
	Envs          []string          `json:"envs,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	TaskSpecs     []*TaskSpec       `json:"tasks"`
	Dependents    []string          `json:"dependents,omitempty"`
	*ResourceSpec `json:"resources,omitempty"`
}

// TaskGroup 表示一组执行命令相同的任务，但每个任务的参数可以不同
type TaskGroup struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Command     string            `json:"command,omitempty"`
	WorkDir     string            `json:"workdir,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Envs        []string          `json:"envs,omitempty"`
	Dependents  []string          `json:"dependents,omitempty"`
	Resources   *ResourceSet      `json:"resources,omitempty"`
	Completions int               `json:"-"`
	Tasks       []*Task           `json:"tasks"`
}

// NewTaskGroupWithSpec 根据指定的TaskGroupSpec内容创建对应的TaskGroup对象
func NewTaskGroupWithSpec(id string, spec *TaskGroupSpec) *TaskGroup {
	group := &TaskGroup{
		ID:          id,
		Name:        spec.Name,
		Command:     strings.ReplaceAll(spec.Command, "\\", "/"),
		WorkDir:     spec.WorkDir,
		Envs:        spec.Envs,
		Labels:      spec.Labels,
		Dependents:  spec.Dependents,
		Resources:   NewResourceSetWithSpec(spec.ResourceSpec),
		Completions: 0,
		Tasks:       make([]*Task, len(spec.TaskSpecs)),
	}
	for i, t := range spec.TaskSpecs {
		group.Tasks[i] = NewTaskWithSpec(group, i, t)
	}
	return group
}

// IsCompleted 判断TaskGroup中所有任务是否都已经成功完成
func (group *TaskGroup) IsCompleted() bool {
	return group.Completions == len(group.Tasks)
}
