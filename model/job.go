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
	// JobTerminating 表示Job正在被终止
	JobTerminating
	// JobTerminated 表示Job被强制终止
	JobTerminated
)

// JobStateToString 将Job的状态转换为字符串
func JobStateToString(state JobState) string {
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
	case JobTerminating:
		return "Terminating"
	case JobTerminated:
		return "Terminated"
	}
	return ""
}

// JobStateFromString 将Job的状态转换为字符串
func JobStateFromString(state string) JobState {
	switch state {
	case "Queued", "queued":
		return JobQueued
	case "Executing", "executing":
		return JobExecuting
	case "Halted", "halted":
		return JobHalted
	case "Completed", "completed":
		return JobCompleted
	case "Failed", "failed":
		return JobFailed
	case "Terminating", "terminating":
		return JobTerminating
	case "Terminated", "terminated":
		return JobTerminated
	}
	return JobQueued
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
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Queue      string            `json:"queue"`
	Priority   int               `json:"priority"`
	Labels     map[string]string `json:"labels"`
	Taints     map[string]string `json:"taints,omitempty"`
	MaxErrors  int               `json:"max_errors"`
	Groups     []*TaskGroup      `json:"groups"`
	SubmitTime time.Time         `json:"submit_time"`
	ExecTime   time.Time         `json:"exec_time"`
	FinishTime time.Time         `json:"finish_time"`
	State      JobState          `json:"state"`
	Progress   int               `json:"progress"`
	TotalTasks int               `json:"-"`
	JSON       []byte            `json:"-"` // 缓存Job的JSON表达
	InitCycle  int64             `json:"-"` // 初次尝试调度的周期
}

// NewJobWithSpec 根据指定的JobSpec内容创建对应的Job对象
func NewJobWithSpec(spec *JobSpec) *Job {
	job := &Job{
		ID:         spec.ID,
		Name:       spec.Name,
		Queue:      spec.Queue,
		Priority:   spec.Priority,
		Labels:     spec.Labels,
		Taints:     spec.Taints,
		MaxErrors:  spec.MaxErrors,
		Groups:     make([]*TaskGroup, len(spec.GroupSpecs)),
		State:      JobQueued,
		Progress:   0,
		TotalTasks: 0}
	if spec.Schedulable == false {
		job.State = JobHalted
	}
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

// GetJSON 获取Job的JSON表达，可指定是否刷新。如果失败返回nil。
func (job *Job) GetJSON(refresh bool) []byte {
	if refresh || job.JSON == nil {
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
	return job.State == JobQueued || job.State == JobExecuting
}

// RefreshState 根据内部任务的状态确定Job的最新状态
func (job *Job) RefreshState() bool {
	if job.State == JobTerminated {
		return false
	}
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
	job.Progress = int(float32(completed) / float32(total) * 100.0)
	if completed == total {
		job.State = JobCompleted
	} else if job.State == JobHalted {
		return false
	} else if waitting == 0 && executing == 0 {
		if job.State == JobTerminating && terminated > 0 {
			job.State = JobTerminated
		} else {
			if terminated > 0 {
				job.State = JobTerminated
			} else {
				job.State = JobFailed
				if job.MaxErrors > 0 && job.MaxErrors >= failed {
					job.State = JobCompleted
				}
			}
		}
	} else if waitting == total {
		job.State = JobQueued
	} else if job.State != JobTerminating {
		job.State = JobExecuting
	}
	if job.State == JobTerminating && terminated > 0 {
		job.State = JobTerminated
	}

	if job.State != JobQueued && job.ExecTime.IsZero() {
		job.ExecTime = time.Now()
	}
	if (job.State == JobCompleted || job.State == JobFailed || job.State == JobTerminated) && job.FinishTime.IsZero() {
		job.FinishTime = time.Now()
	}
	return last != job.State
}

type JobSortField int

const (
	SortJobByDefault JobSortField = iota
	SortJobByState
	SortJobBySubmit
)

// GeneralJobSorter 是Job通常情况下的排序方式
type GeneralJobSorter struct {
	Jobs   []*Job
	SortBy JobSortField
}

func (s *GeneralJobSorter) Len() int {
	return len(s.Jobs)
}

func (s *GeneralJobSorter) Less(i, j int) bool {
	if s.SortBy == SortJobByState {
		if s.Jobs[i].State < s.Jobs[j].State {
			return true
		} else if s.Jobs[i].State > s.Jobs[j].State {
			return false
		}
		if s.Jobs[i].Priority > s.Jobs[j].Priority {
			return true
		} else if s.Jobs[i].Priority < s.Jobs[j].Priority {
			return false
		}
		return s.Jobs[i].SubmitTime.Before(s.Jobs[j].SubmitTime)
	} else if s.SortBy == SortJobBySubmit {
		return s.Jobs[i].SubmitTime.After(s.Jobs[j].SubmitTime)
	} else {
		if s.Jobs[i].IsSchedulable() && !s.Jobs[j].IsSchedulable() {
			return true
		} else if !s.Jobs[i].IsSchedulable() && s.Jobs[j].IsSchedulable() {
			return false
		}
		if s.Jobs[i].Priority > s.Jobs[j].Priority {
			return true
		} else if s.Jobs[i].Priority < s.Jobs[j].Priority {
			return false
		}
		return s.Jobs[i].SubmitTime.Before(s.Jobs[j].SubmitTime)
	}
}

func (s *GeneralJobSorter) Swap(i, j int) {
	s.Jobs[i], s.Jobs[j] = s.Jobs[j], s.Jobs[i]
}

// JobUpdatableProps 包含Job在提交后可以修改的属性
type JobUpdatableProps struct {
	Name      string            `json:"name,omitempty"`
	Priority  *int              `json:"priority,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Taints    map[string]string `json:"taints,omitempty"`
	MaxErrors *int              `json:"max_errors,omitempty"`
}
