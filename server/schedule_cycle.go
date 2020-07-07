package server

import (
	"log"
	"sync/atomic"

	"github.com/qianxiaoming/lightsched/model"
)

type scheduleRecord struct {
	job  *model.Job
	task *model.Task
	node string
}

func scheduleCycle(svc *APIServer) []scheduleRecord {
	// 使用读锁保护内部数据。在调度过程中实际要修改的是Task的状态，记录它被
	// 调度到哪个节点，但是这些信息可以先记录到局部变量中，避免长时间持有写锁
	svc.state.RLock()
	defer svc.state.RUnlock()
	// 获取所有可以调度的JobQueue，按照优先级排序
	queues := svc.state.GetSchedulableQueues()
	if len(queues) == 0 {
		log.Println("No schedulable job queue found. Stop schedule cycle.")
		return nil
	}

	// 使用1个切片保存此次所有成功调度的Task
	scheduleTable := make([]scheduleRecord, 0, 64)
	for _, curQueue := range queues {
		// 获取所有可以调度的Job，按照优先级、时间和初次调度周期排序
		jobs := curQueue.GetSchedulableJobs()
		for {
			// 确定当前可调度Job的最高优先级
			maxPriority := 0
			for p := range jobs {
				if maxPriority < p {
					maxPriority = p
				}
			}
			// 公平调度优先级为maxPriority的多个Job。首先获取它们各自可以调度的Task，然后
			// 使用无限循环反复调度，直到没有Task可以调度为止（可能是所有Task都被调度，也可
			// 能是没有节点可以接受任何Task）
			jobTasks := make([][]*model.Task, len(jobs[maxPriority]))
			for i, job := range jobs[maxPriority] {
				jobTasks[i] = job.GetSchedulableTasks()
			}
			for {
				count := len(scheduleTable)
				for i := 0; i < len(jobTasks); i++ {
					for _, task := range jobTasks[i] {
						if task.State != model.TaskQueued {
							continue
						}
						// 尝试调度task到某个节点上

						scheduleTable = append(scheduleTable, scheduleRecord{task: task, node: ""})
					}
				}
				// 如果没有任何Task被成功调度，跳出此次循环
				if count == len(scheduleTable) {
					break
				}
			}
			// 当前最高优先级的Job都已经无法调度。移除它们后进入下一次循环。
			delete(jobs, maxPriority)
			if len(jobs) == 0 {
				break
			}
		}
	}
	return scheduleTable
}

func (svc *APIServer) runScheduleCycle() {
	// 检查调度标志是否被设置
	svc.schedCycle = svc.schedCycle + 1
	flag := atomic.SwapInt32(&svc.schedFlag, 0)
	if flag == 0 {
		return
	}
	log.Printf("Run schedule cycle %v...\n", svc.schedCycle)

	scheduleTable := scheduleCycle(svc)
	if len(scheduleTable) == 0 {
		return
	}

	// 修改被调度的Task状态。此时使用写锁保护，但要注意Job或Task数据可能已经被修改
	svc.state.Lock()
	defer svc.state.Unlock()
	reschedule := false
	for _, record := range scheduleTable {
		if record.job.State == model.JobQueued || record.job.State == model.JobExecuting {

		} else {
			// Job的状态已经发生变化，该Job的所有Task无需调度，所以需要重新执行一次调度
			reschedule = true
		}
	}
	if reschedule {
		svc.setScheduleFlag()
	}
}
