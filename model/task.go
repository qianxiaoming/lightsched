package model

import "time"

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

// Task 是具体执行的计算任务
type Task struct {
	ID         string
	Name       string
	Envs       []string
	Command    string
	Args       string
	WorkDir    string
	Labels     map[string]string
	Resources  ResourceSet
	State      TaskState
	Progress   int
	ExitCode   int
	ExecNode   string
	StartTime  time.Time
	FinishTime time.Time
}

// TaskGroup 表示一组执行命令相同的任务，但每个任务的参数可以不同
type TaskGroup struct {
	Command    string
	WorkDir    string
	Envs       map[string]string
	Labels     map[string]string
	Tasks      []*Task
	Dependents []*TaskGroup
	Resources  ResourceSet
}

// TaskGroupSpec 表示提交的任务组的基本信息，其中包含多个任务描述。
type TaskGroupSpec struct {
	Count     int
	Command   string
	WorkDir   string
	Envs      map[string]string
	Labels    map[string]string
	Resources ResourceSetSpec
}
