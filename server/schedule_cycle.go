package server

import (
	"encoding/json"
	"log"
	"sort"
	"sync/atomic"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
)

// scheduleRecord 表示可调度的节点
type scheduleNode struct {
	node      *model.WorkNode
	available *model.ResourceSet
	score     float32
}

// scheduleRecord 记录了1个调度结果，即哪个任务调度到哪个节点
type scheduleRecord struct {
	task   *model.Task
	target *scheduleNode
}

// 定义该类用以排序计算任务：总是把GPU任务放在前面
type taskSlice []*model.Task

// sort包要求的排序接口实现
func (p taskSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p taskSlice) Len() int           { return len(p) }
func (p taskSlice) Less(i, j int) bool { return p[i].Resources.GPU.Cards >= p[j].Resources.GPU.Cards }

// scheduleOneTask 尝试将1个任务调度到某个节点上，若无法调度返回nil
func scheduleOneTask(svc *APIServer, task *model.Task, nodes []*scheduleNode) *scheduleNode {
	var target *scheduleNode = nil
	var maxScore float32 = 0.0
	// 遍历节点列表计算Task在该节点上的得分
	for _, node := range nodes {
		// 检查label和taints是否符合
		passed := true
		for k, v := range task.Labels {
			nv, ok := node.node.Labels[k]
			if !ok || nv != v {
				passed = false
				if svc.schedLog {
					log.Printf("  Task %s failed scheduling to %s because of label %s", task.ID, node.node.Name, k)
				}
				break
			}
		}
		if !passed {
			continue
		}
		passed = true
		for k, v := range task.Taints {
			nv, ok := node.node.Taints[k]
			if ok && nv == v {
				passed = false
				if svc.schedLog {
					log.Printf("  Task %s failed scheduling to %s because of taints %s", task.ID, node.node.Name, k)
				}
				break
			}
		}
		if !passed {
			continue
		}
		// 检查资源是否符合并算分
		node.score = 0.0
		if ok, res, need, offered := task.Resources.SatisfiedWith(node.available); ok {
			// 计算CPU和内存得分
			if task.Resources.CPU.Cores > 0 {
				node.score = (node.available.CPU.Cores / node.node.Resources.CPU.Cores) * 3.0
			} else {
				node.score = (float32(node.available.CPU.Frequency) / float32(node.node.Resources.CPU.Frequency)) * 3.0
			}
			node.score = node.score * float32(node.node.Resources.CPU.MinFreq) / 2400.0
			node.score = node.score + float32(node.available.Memory)/float32(node.node.Resources.Memory)
			if task.Resources.GPU.Cards > 0 {
				// 计算GPU得分
				gpuScore := (float32(node.available.GPU.Cards) / float32(node.node.Resources.GPU.Cards)) * 5.0
				gpuScore = gpuScore * float32(node.node.Resources.GPU.Memory) / 8.0
				node.score += gpuScore
			}
			// 记录当前得分最高的节点
			if maxScore < node.score {
				maxScore = node.score
				target = node
			}
		} else {
			if svc.schedLog {
				log.Printf("  Task %s failed scheduling to %s: %s need %v but %v offered", task.ID, node.node.Name, res, need, offered)
			}
		}
	}
	if target != nil && svc.schedLog {
		log.Printf("  Task %s is scheduled to %s with score %f", task.ID, target.node.Name, maxScore)
	}
	return target
}

// scheduleCycle 实现一个调度周期，返回此次周期内的调度结果
func scheduleCycle(svc *APIServer) []scheduleRecord {
	// 获取所有可以调度的JobQueue，按照优先级排序
	queues := svc.state.GetSchedulableQueues()
	if len(queues) == 0 {
		log.Println("No schedulable job queue found. Stop schedule cycle.")
		return nil
	}

	// 获取所有可以调度的节点
	scheduleNodes := make([]*scheduleNode, 0, len(svc.nodes.GetNodes()))
	for _, node := range svc.nodes.GetNodes() {
		if node.State == model.NodeOnline {
			n := &scheduleNode{node: node, available: (&node.Available).Clone()}
			scheduleNodes = append(scheduleNodes, n)
		}
	}
	if len(scheduleNodes) == 0 {
		log.Println("No schedulable node found. Stop schedule cycle.")
		return nil
	}

	// 使用1个切片保存此次所有成功调度的Task
	scheduleTable := make([]scheduleRecord, 0, 64)
	for _, curQueue := range queues {
		// 获取所有可以调度的Job，按照优先级、时间和初次调度周期排序
		jobs := curQueue.GetSchedulableJobs()
		for len(jobs) > 0 {
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
			jobTasks := make([]taskSlice, len(jobs[maxPriority]))
			for i, job := range jobs[maxPriority] {
				jobTasks[i] = job.GetSchedulableTasks()
				sort.Sort(jobTasks[i])
			}
			for i := 0; i < len(jobTasks); i++ {
				for _, task := range jobTasks[i] {
					// 尝试调度task到某个节点上
					target := scheduleOneTask(svc, task, scheduleNodes)
					if target != nil {
						scheduleTable = append(scheduleTable, scheduleRecord{task: task, target: target})
						// 从节点的可用资源中减去Task所需的资源
						target.available.Consume(task.Resources)
					}
				}
			}
			// 当前最高优先级的Job都已经无法调度。移除它们后进入下一次循环。
			delete(jobs, maxPriority)
		}
	}
	return scheduleTable
}

func (svc *APIServer) runScheduleCycle() {
	// 检查调度标志是否被设置
	flag := atomic.SwapInt32(&svc.schedFlag, 0)
	if flag == 0 {
		return
	}
	svc.schedCycle = svc.schedCycle + 1

	svc.state.Lock()
	defer svc.state.Unlock()

	svc.nodes.Lock()
	defer svc.nodes.Unlock()

	// 执行调度，获得调度结果表
	log.Printf("Run schedule cycle %d\n", svc.schedCycle)
	scheduleTable := scheduleCycle(svc)
	if len(scheduleTable) == 0 {
		log.Println("There is no task scheduled in this cycle")
		return
	}
	log.Printf("There are %d task(s) scheduled in this cycle", len(scheduleTable))

	// 修改被调度的Task状态
	updates := make([]*model.Task, 0, len(scheduleTable))
	for _, record := range scheduleTable {
		record.task.State = model.TaskScheduled
		record.task.NodeName = record.target.node.Name
		record.target.node.Available.Consume(record.task.Resources)
		// 缓存调度结果，以便节点拉取调度到自身的Task
		msg, _ := json.Marshal(record.task)
		svc.nodes.AppendNodeMessage(record.target.node.Name, message.KindScheduleTask, record.task.ID, msg)

		updates = append(updates, record.task)
	}
	// 更新数据库中Task的状态
	if err := svc.state.SaveTasks(updates); err != nil {
		log.Printf("Unable to save tasks into database: %v", err)
	}

	log.Println("Schedule cycle done")
}
