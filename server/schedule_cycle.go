package server

import (
	"log"
	"sync/atomic"

	"github.com/qianxiaoming/lightsched/model"
)

func (svc *APIServer) runScheduleCycle() {
	// 检查调度标志是否被设置
	svc.schedCycle = svc.schedCycle + 1
	flag := atomic.SwapInt32(&svc.schedFlag, 0)
	if flag == 0 {
		return
	}
	log.Printf("Run schedule cycle %v...\n", svc.schedCycle)

	// 获取所有可以调度的JobQueue，按照优先级排序
	svc.state.Lock()
	defer svc.state.Unlock()
	queues := svc.state.GetOrderedJobQueues()
	if len(queues) == 0 {
		log.Println("No schedulable job queue found. Stop schedule cycle.")
		return
	}
	// 使用1个切片保存此次所有成功调度的Task
	scheduled := make([]*model.Task, 0, 64)
	for i := len(queues) - 1; i >= 0; i-- {
		curQueue := queues[i]
		// 获取所有可以调度的Job，按照优先级、时间和初次调度周期排序
		jobs := curQueue.GetOrderedJobs()
		for {
			// 确定当前可调度Job的最高优先级
			maxPriority := 0
			for p := range jobs {
				if maxPriority < p {
					maxPriority = p
				}
			}
			// 优先调度优先级最高的Job。如果有多个Job优先级相同，则依次调度它们的Task
			// 使用无限循环反复调度这个队列中的Job，直到没有Task可以调度为止
			schedTasks := make([][]*model.Task, len(jobs[maxPriority]))
			for i, job := range jobs[maxPriority] {
				schedTasks[i] = job.GetSchedulableTasks()
			}
			for {
				count := len(scheduled)
				for i := 0; i < len(schedTasks); i++ {
					for j := 0; j < len(schedTasks[i]); j++ {
						t := schedTasks[i][j]
						if t.State != model.TaskQueued {
							continue
						}
						// 尝试调度t到某个节点上
						t.State = model.TaskScheduled
						scheduled = append(scheduled, t)
					}
				}
				// 如果没有任何Task被成功调度，跳出此次循环
				if count == len(scheduled) {
					break
				}
			}
			// 当前最高优先级的Job已经无法调度
			delete(jobs, maxPriority)
			if len(jobs) == 0 {
				break
			}
		}
	}
}
