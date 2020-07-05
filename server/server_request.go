package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
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

	// 创建Job需要的目录。以下功能如果失败不影响任务的执行，因此仅输出日志并不返回失败。
	dir := filepath.Join(svc.config.dataPath, job.ID)
	if err := util.MakeDirAll(dir); err != nil {
		return err
	}
	b, err := json.MarshalIndent(job, "", "    ")
	if err == nil {
		err = ioutil.WriteFile(filepath.Join(dir, "job_content.json"), b, 0666)
		if err != nil {
			log.Printf("Unable to write job content under %s: %v", dir, err)
		}
	} else {
		log.Printf("Failed to marshal job \"%s\" to JSON: %v", job.Name, err)
	}

	return nil
}
