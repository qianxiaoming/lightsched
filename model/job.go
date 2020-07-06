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
	// JobWaiting 表示Job正在等待执行
	JobWaiting
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

// JobSpec 表示提交的作业的基本信息，包含多个任务组的描述。
type JobSpec struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Queue       string            `json:"queue"`
	Priority    int               `json:"priority,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
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
}

// NewJobWithSpec 根据指定的JobSpec内容创建对应的Job对象
func NewJobWithSpec(spec *JobSpec) *Job {
	job := &Job{
		ID:          spec.ID,
		Name:        spec.Name,
		Queue:       spec.Queue,
		Priority:    spec.Priority,
		Labels:      spec.Labels,
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
