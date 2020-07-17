package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
)

func (svc *APIServer) requestCreateJob(spec *model.JobSpec) error {
	// 如果没有指定作业编号和队列则指定默认值
	if len(spec.ID) == 0 {
		spec.ID = util.GenerateUUID()
	}
	if len(spec.Queue) == 0 {
		spec.Queue = constant.DefaultQueueName
	}

	// 创建Job对象并生成TaskGroup及Task对象，保存到服务状态数据中
	job := model.NewJobWithSpec(spec)
	if err := func() error {
		svc.state.Lock()
		defer svc.state.Unlock()
		return svc.state.AddJob(job)
	}(); err != nil {
		return err
	}

	// 标记任务调度状态
	svc.setScheduleFlag()

	// 创建Job需要的目录。以下功能如果失败不影响任务的执行，因此仅输出日志并不返回失败。
	dir := filepath.Join(svc.config.dataPath, job.ID)
	if err := util.MakeDirAll(dir); err != nil {
		log.Printf("Unable to create job directory %s: %v", dir, err)
	} else {
		if err := ioutil.WriteFile(filepath.Join(dir, "job_content.json"), job.GetJSON(false), 0666); err != nil {
			log.Printf("Unable to write job content under %s: %v", dir, err)
		}
		job.JSON = nil
	}

	return nil
}

func (svc *APIServer) requestRegisterNode(ip string, req *message.RegisterNode) error {
	if len(req.Name) == 0 {
		return fmt.Errorf("the name of the node is empty")
	}
	node := &model.WorkNode{
		Name:      req.Name,
		Address:   ip,
		Platform:  req.Platform,
		State:     model.NodeOnline,
		Online:    time.Now(),
		Labels:    req.Labels,
		Taints:    nil,
		Resources: req.Resources,
		Reserved:  *model.DefaultResourceSet,
		Available: req.Resources,
	}
	node.Available.Consume(&node.Reserved)

	svc.nodes.Lock()
	defer svc.nodes.Unlock()
	svc.nodes.AddNode(node)

	// 标记任务调度状态
	svc.setScheduleFlag()
	return nil
}

func (svc *APIServer) requestUpdateTasks(updates []*message.TaskStatus) {
	svc.state.Lock()
	defer svc.state.Unlock()
	svc.nodes.Lock()
	defer svc.nodes.Unlock()

	reschedule := false
	// 更新Task及对应Job的状态
	for _, update := range updates {
		task := svc.state.UpdateTaskStatus(update.ID, update.State, update.Progress, update.ExitCode, update.Error)
		if task != nil && model.IsFinishState(update.State) {
			reschedule = true
			// 归还Task消耗的节点资源
			node := svc.nodes.GetNode(task.NodeName)
			if node != nil {
				node.Available.GiveBack(task.Resources)
			}
		}
	}

	if reschedule {
		svc.setScheduleFlag()
	}
}

func (svc *APIServer) requestTerminateJob(id string) error {
	svc.state.Lock()
	defer svc.state.Unlock()
	log.Printf("Terminate Job %s\n", id)
	if err := svc.state.SetJobState(id, model.JobTerminating); err != nil {
		return err
	}
	job := svc.state.GetJob(id)
	// 确定所有运行此Job的节点名字
	nodes := make(map[string]bool)
	for _, g := range job.Groups {
		for _, t := range g.Tasks {
			if t.State != model.TaskExecuting && t.State != model.TaskDispatching && t.State != model.TaskScheduled {
				continue
			}
			nodes[t.NodeName] = true
		}
	}

	svc.nodes.Lock()
	defer svc.nodes.Unlock()
	for name := range nodes {
		svc.nodes.AppendNodeMessage(name, message.KindTerminateJob, id, nil)
	}
	return nil
}
