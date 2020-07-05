package model

import (
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
	JobSpec
	Groups     []*TaskGroup
	SubmitTime time.Time
	ExecTime   time.Time
	FinishTime time.Time
	State      JobState
	Progress   int
}

// NewJobWithSpec 根据指定的JobSpec内容创建对应的Job对象
func NewJobWithSpec(spec *JobSpec) *Job {
	job := &Job{
		JobSpec:    *spec,
		Groups:     make([]*TaskGroup, len(spec.GroupSpecs)),
		SubmitTime: time.Now(),
		State:      JobQueued,
		Progress:   0}
	for i, g := range spec.GroupSpecs {
		job.Groups[i] = NewTaskGroupWithSpec(fmt.Sprintf("%s.%d", job.ID, i), g)
	}
	return job
}
