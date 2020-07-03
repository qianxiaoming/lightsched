package server

import (
	"sync"

	"github.com/qianxiaoming/lightsched/common"
)

// StateModel 是API Server的内部状态数据
type StateModel struct {
	sync.RWMutex
	jobQueues map[string]*common.JobQueue
	jobMap    map[string]*common.Job
	jobList   []*common.Job
}

// NewStateModel 创建服务的内部状态数据对象
func NewStateModel() *StateModel {
	return &StateModel{
		jobQueues: make(map[string]*common.JobQueue),
		jobMap:    make(map[string]*common.Job),
		jobList:   make([]*common.Job, 0, 128),
	}
}

func (m *StateModel) getJobQueue(name string) *common.JobQueue {
	queue, ok := m.jobQueues[name]
	if ok {
		return queue
	}
	return nil
}

func (m *StateModel) getJob(id string) *common.Job {
	job, ok := m.jobMap[id]
	if ok {
		return job
	}
	return nil
}
