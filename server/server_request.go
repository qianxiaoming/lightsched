package server

import (
	"path/filepath"

	"github.com/qianxiaoming/lightsched/data"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
	uuid "github.com/satori/go.uuid"
)

func (svc *APIServer) requestCreateJob(spec *model.JobSpec) error {
	// 如果没有指定作业编号和队列则指定默认值
	if len(spec.ID) == 0 {
		spec.ID = uuid.NewV4().String()
	}
	if len(spec.Queue) == 0 {
		spec.Queue = data.DefaultQueueName
	}

	// 创建Job对象并生成TaskGroup及Task对象，保存到服务状态数据中
	job := model.NewJobWithSpec(spec)
	if err := svc.state.AppendJob(job); err != nil {
		return err
	}
	// 标记任务调度状态
	svc.setScheduleFlag()

	// 创建Job需要的目录
	dir := filepath.Join(svc.config.dataPath, job.ID)
	if err := util.MakeDirAll(dir); err != nil {
		return err
	}

	return nil
}
