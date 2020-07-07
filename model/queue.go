package model

import "sort"

// JobQueueSpec 是描述了JobQueue的信息
type JobQueueSpec struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
}

// JobQueue 是可包含多个作业的集合队列
type JobQueue struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Jobs     []*Job
}

// NewJobQueueWithSpec 创建新的JobQueue对象
func NewJobQueueWithSpec(spec *JobQueueSpec) *JobQueue {
	return &JobQueue{
		Name:     spec.Name,
		Enabled:  spec.Enabled,
		Priority: spec.Priority,
		Jobs:     make([]*Job, 0, 1),
	}
}

type JobQueueSlice []*JobQueue

func (s JobQueueSlice) Len() int {
	return len(s)
}

func (s JobQueueSlice) Less(i, j int) bool {
	// 优先级高的排在前面
	return s[i].Priority > s[j].Priority
}

func (s JobQueueSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// GetSchedulableJobs 获取按照优先级、时间和初次调度周期排序后的作业集合
func (queue *JobQueue) GetSchedulableJobs() map[int][]*Job {
	// 将可调度的Job按照优先级放到不同的队列中
	jobs := make(map[int][]*Job, 1)
	for _, j := range queue.Jobs {
		if !j.IsSchedulable() {
			continue
		}
		if _, ok := jobs[j.Priority]; !ok {
			jobs[j.Priority] = make([]*Job, 0, 16)
		}
		jobs[j.Priority] = append(jobs[j.Priority], j)
	}
	for _, v := range jobs {
		sort.Sort(JobSlice(v))
	}
	return jobs
}
