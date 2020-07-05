package model

// JobQueueSpec 是描述了JobQueue的信息
type JobQueueSpec struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
}

// JobQueue 是可包含多个作业的集合队列
type JobQueue struct {
	JobQueueSpec
	Jobs []*Job
}

// NewJobQueueWithSpec 创建新的JobQueue对象
func NewJobQueueWithSpec(spec *JobQueueSpec) *JobQueue {
	return &JobQueue{
		JobQueueSpec: *spec,
		Jobs:         make([]*Job, 0, 1),
	}
}
