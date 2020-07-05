package model

// JobQueue 是可包含多个作业的队列
type JobQueue struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Jobs     []*Job `json: "-"`
}

// NewJobQueue 创建新的JobQueue对象
func NewJobQueue(name string, enabled bool, priority int) *JobQueue {
	return &JobQueue{
		Name:     name,
		Enabled:  enabled,
		Priority: priority,
		Jobs:     make([]*Job, 0, 1),
	}
}
