package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// JobState 表示Job的状态
type JobState int32

const (
	// JobQueued 表示Job已经提交并入队
	JobQueued JobState = iota
	// JobExecuting 表示Job中的任务正在执行
	JobExecuting
	// JobHalted 表示Job暂停执行，不调度其中未执行的任务
	JobHalted
	// JobCompleted 表示Job中的所有任务都成功执行
	JobCompleted
	// JobFailed 表示Job中的失败任务数超过了设定值
	JobFailed
	// JobTerminated 表示Job被强制终止
	JobTerminated
)

func JobStateString(state JobState) string {
	switch state {
	case JobQueued:
		return "Queued"
	case JobExecuting:
		return "Executing"
	case JobHalted:
		return "Halted"
	case JobCompleted:
		return "Completed"
	case JobFailed:
		return "Failed"
	case JobTerminated:
		return "Terminated"
	}
	return ""
}

// JobSpec 表示提交的作业的基本信息，包含多个任务组的描述。
type JobSpec struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Queue       string            `json:"queue"`
	Priority    int               `json:"priority,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Taints      map[string]string `json:"taints,omitempty"`
	Schedulable bool              `json:"schedulable"`
	MaxErrors   int               `json:"max_errors,omitempty"`
	GroupSpecs  []*TaskGroupSpec  `json:"groups"`
}

// Job 表示要执行的多个任务组集合。任务组之间可以有依赖关系。
type Job struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Queue       string            `json:"queue"`
	Priority    int               `json:"priority"`
	Labels      map[string]string `json:"labels"`
	Taints      map[string]string `json:"taints,omitempty"`
	Schedulable bool              `json:"schedulable"`
	MaxErrors   int               `json:"max_errors"`
	Groups      []*TaskGroup      `json:"groups"`
	SubmitTime  time.Time         `json:"submit_time"`
	ExecTime    time.Time         `json:"exec_time"`
	FinishTime  time.Time         `json:"finish_time"`
	State       JobState          `json:"state"`
	Progress    int               `json:"-"`
	TotalTasks  int               `json:"-"`
	JSON        []byte            `json:"-"` // 缓存Job的JSON表达
	InitCycle   int64             `json:"-"` // 初次尝试调度的周期
}

// NewJobWithSpec 根据指定的JobSpec内容创建对应的Job对象
func NewJobWithSpec(spec *JobSpec) *Job {
	job := &Job{
		ID:          spec.ID,
		Name:        spec.Name,
		Queue:       spec.Queue,
		Priority:    spec.Priority,
		Labels:      spec.Labels,
		Taints:      spec.Taints,
		Schedulable: spec.Schedulable,
		MaxErrors:   spec.MaxErrors,
		Groups:      make([]*TaskGroup, len(spec.GroupSpecs)),
		State:       JobQueued,
		Progress:    0,
		TotalTasks:  0}
	for i, g := range spec.GroupSpecs {
		job.Groups[i] = NewTaskGroupWithSpec(fmt.Sprintf("%s.%d", job.ID, i), g)
	}
	return job
}

// CountTasks 计算Job包含的任务总数
func (job *Job) CountTasks() int {
	if job.TotalTasks == 0 {
		for _, g := range job.Groups {
			job.TotalTasks = job.TotalTasks + len(g.Tasks)
		}
	}
	return job.TotalTasks
}

// GetJSON 获取Job的JSON表达。如果失败返回nil。
func (job *Job) GetJSON() []byte {
	if job.JSON == nil {
		if b, err := json.Marshal(job); err == nil {
			job.JSON = b
		} else {
			panic(err)
		}
	}
	return job.JSON
}

// GetTaskGroup 返回Job中指定名称的TaskGroup
func (job *Job) GetTaskGroup(name string) *TaskGroup {
	for _, group := range job.Groups {
		if group.Name == name {
			return group
		}
	}
	return nil
}

// GetSchedulableTasks 返回Job中当前可以调度的所有Task
func (job *Job) GetSchedulableTasks() []*Task {
	var tasks []*Task = nil
	for _, group := range job.Groups {
		// 首先判定前置的TaskGroup是否都完成了
		schedulable := true
		for _, dependent := range group.Dependents {
			g := job.GetTaskGroup(dependent)
			if g != nil && !g.IsCompleted() {
				schedulable = false
				break
			}
		}
		if !schedulable {
			continue
		}
		for _, task := range group.Tasks {
			if task.State == TaskQueued {
				if tasks == nil {
					tasks = make([]*Task, 0, 16)
				}
				tasks = append(tasks, task)
			}
		}
	}
	return tasks
}

// IsSchedulable 判断Job是否可以被调度
func (job *Job) IsSchedulable() bool {
	return job.Schedulable && (job.State == JobQueued || job.State == JobExecuting)
}

// RefreshState 根据内部任务的状态确定Job的最新状态
func (job *Job) RefreshState() bool {
	last := job.State
	total := job.CountTasks()
	var waitting, executing, completed, failed, terminated int
	for _, g := range job.Groups {
		for _, t := range g.Tasks {
			switch t.State {
			case TaskQueued:
				waitting++
			case TaskScheduled, TaskDispatching, TaskExecuting:
				executing++
			case TaskCompleted:
				completed++
			case TaskFailed, TaskAborted:
				failed++
			case TaskTerminated:
				terminated++
			}
		}
	}
	if completed == total {
		job.State = JobCompleted
	} else if waitting == 0 && executing == 0 {
		if terminated > 0 {
			job.State = JobTerminated
		} else {
			job.State = JobFailed
			if job.MaxErrors > 0 && job.MaxErrors >= failed {
				job.State = JobCompleted
			}
		}
	} else if waitting == total {
		job.State = JobQueued
	} else {
		job.State = JobExecuting
	}
	if job.State != JobQueued && job.ExecTime.IsZero() {
		job.ExecTime = time.Now()
	}
	if (job.State == JobCompleted || job.State == JobFailed || job.State == JobTerminated) && job.FinishTime.IsZero() {
		job.FinishTime = time.Now()
	}
	return last != job.State
}

// GeneralJobSlice 是Job通常情况下的排序方式
type GeneralJobSlice []*Job

func (s GeneralJobSlice) Len() int {
	return len(s)
}

func (s GeneralJobSlice) Less(i, j int) bool {
	if s[i].IsSchedulable() && !s[j].IsSchedulable() {
		return true
	} else if !s[i].IsSchedulable() && s[j].IsSchedulable() {
		return false
	}
	if s[i].Priority > s[j].Priority {
		return true
	} else if s[i].Priority < s[j].Priority {
		return false
	}
	if s[i].SubmitTime.Before(s[j].SubmitTime) {
		return true
	} else if s[i].SubmitTime.After(s[j].SubmitTime) {
		return false
	}
	return s[i].Name < s[j].Name
}

func (s GeneralJobSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
