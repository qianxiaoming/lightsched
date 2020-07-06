package server

import (
	"log"
	"sync/atomic"
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
	queues := svc.state.GetOrderedJobQueues()
	if len(queues) == 0 {
		log.Println("No schedulable job queue found. Stop schedule cycle.")
		return
	}
	for i := len(queues) - 1; i >= 0; i-- {
		curQueue := queues[i]
		// 获取所有可以调度的Job，按照优先级、时间和初次调度周期排序
		jobs := curQueue.GetOrderedJobs()
		maxPriority := 0
		for p := range jobs {
			if maxPriority < p {
				maxPriority = p
			}
		}
		// 优先调度在此队列中优先级最高的Job
		for _, job := range jobs[maxPriority] {
			log.Printf("  Schedule job \"%s\"...\n", job.Name)
		}
	}
}
