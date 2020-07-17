package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
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
		Resources: (&req.Resources).Clone(),
		Reserved:  model.DefaultResourceSet,
		Available: (&req.Resources).Clone(),
	}
	node.Available.Consume(node.Reserved)

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
	log.Printf("Terminating Job %s\n", id)
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

func (svc *APIServer) requestListJobs(filterState *model.JobState, sortField model.JobSortField, offset, limits int) []*message.JobInfo {
	svc.state.Lock()
	defer svc.state.Unlock()

	allJobs := svc.state.GetJobList(filterState, sortField, offset, limits)
	if allJobs == nil {
		return nil
	}

	infos := make([]*message.JobInfo, 0, len(allJobs))
	for _, j := range allJobs {
		infos = append(infos, message.NewJobInfo(j))
	}
	return infos
}

func (svc *APIServer) requestGetJob(jobid string) *message.JobInfo {
	svc.state.Lock()
	defer svc.state.Unlock()

	if job := svc.state.GetJob(jobid); job != nil {
		return message.NewJobInfo(job)
	}
	return nil
}

func (svc *APIServer) requestDeleteJob(jobid string) error {
	svc.state.Lock()
	defer svc.state.Unlock()

	log.Printf("Deleting Job %s...\n", jobid)
	if err := svc.state.DeleteJob(jobid); err != nil {
		return err
	}
	dir := filepath.Join(svc.config.dataPath, jobid)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	log.Println("Job deleted")
	return nil
}

func (svc *APIServer) requestListNodes() []*message.NodeInfo {
	svc.nodes.Lock()
	defer svc.nodes.Unlock()

	nodes := svc.nodes.GetNodes()
	if len(nodes) == 0 {
		return nil
	}
	infos := make([]*message.NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		info := &message.NodeInfo{
			Name:      n.Name,
			Address:   n.Address,
			Platform:  n.Platform,
			State:     n.State,
			Online:    n.Online,
			Labels:    util.CloneMap(n.Labels),
			Taints:    util.CloneMap(n.Taints),
			Resources: n.Resources.Clone(),
			Reserved:  n.Reserved.Clone(),
			Available: n.Available.Clone(),
		}
		infos = append(infos, info)
	}
	return infos
}

func (svc *APIServer) requestGetTask(id string) *message.TaskInfo {
	svc.state.Lock()
	defer svc.state.Unlock()

	jobid, groupid, taskid := model.ParseTaskID(id)
	job := svc.state.GetJob(jobid)
	if job == nil {
		return nil
	}
	task := job.Groups[groupid].Tasks[taskid]
	return message.NewTaskInfo(task)
}

func (svc *APIServer) requestGetJobTasks(id string) []*message.TaskBriefInfo {
	svc.state.Lock()
	defer svc.state.Unlock()

	job := svc.state.GetJob(id)
	if job == nil {
		return nil
	}
	infos := make([]*message.TaskBriefInfo, 0, job.CountTasks())
	for _, group := range job.Groups {
		for _, task := range group.Tasks {
			infos = append(infos, message.NewTaskBriefInfo(task))
		}
	}
	return infos
}

func (svc *APIServer) requestGetTaskBrief(id string) *message.TaskBriefInfo {
	svc.state.Lock()
	defer svc.state.Unlock()

	jobid, groupid, taskid := model.ParseTaskID(id)
	job := svc.state.GetJob(jobid)
	if job == nil {
		return nil
	}
	task := job.Groups[groupid].Tasks[taskid]
	return message.NewTaskBriefInfo(task)
}

func (svc *APIServer) requestGetTaskLog(id string) io.ReadCloser {
	svc.state.Lock()
	defer svc.state.Unlock()

	jobid, groupid, taskid := model.ParseTaskID(id)
	job := svc.state.GetJob(jobid)
	if job == nil {
		return nil
	}
	filename := filepath.Join(svc.config.dataPath, jobid, fmt.Sprintf("%d.%d.log", groupid, taskid))
	file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		log.Printf("Unable to open log file %s: %v\n", filename, err)
		return nil
	}
	return file
}
