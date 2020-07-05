package common

import (
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
	ID          string
	Name        string
	Queue       string
	Priority    int
	Labels      map[string]string
	Schedulable bool
	MaxErrors   int
	GroupSpecs  []TaskGroupSpec
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

	return job
}
