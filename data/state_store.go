package data

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
	bolt "go.etcd.io/bbolt"
)

var (
	// DatabaseBuckets 是数据库中存储数据单元的默认名称
	// config: 系统的配置信息
	// queue: 作业队列信息
	// job: 所有Job信息（包含已经完成的）
	// task: 所有计算任务信息。计算任务的唯一标识包含所属Job的标识，使用:分隔（便于前缀遍历）
	DatabaseBuckets = [4]string{"config", "queue", "job", "task"}
)

// StateStore 是API Server的内部状态数据
type StateStore struct {
	sync.RWMutex
	dbPath    string
	boltDB    *BoltDB
	jobQueues map[string]*model.JobQueue
	jobMap    map[string]*model.Job
	jobList   []*model.Job
}

// NewStateStore 创建服务的内部状态数据对象
func NewStateStore() *StateStore {
	return &StateStore{
		jobQueues: make(map[string]*model.JobQueue, 1),
		jobMap:    make(map[string]*model.Job, 128),
		jobList:   make([]*model.Job, 0, 128),
	}
}

func (m *StateStore) InitState(path string) error {
	m.dbPath = filepath.Join(path, constant.DatabaseFileName)
	if !util.PathExists(m.dbPath) {
		if err := createDatabaseFile(m.dbPath); err != nil {
			return err
		}
	}
	if err := m.loadFromDatabase(); err != nil {
		return err
	}
	return nil
}

func (m *StateStore) ClearState() {
	if m.boltDB != nil {
		log.Println("Close database file")
		m.boltDB.Close()
	}
	m.boltDB = nil
}

func createDatabaseFile(dbfile string) error {
	log.Printf("Creating database file \"%s\" on first startup...", dbfile)
	if err := util.MakeDirAll(filepath.Dir(dbfile)); err != nil {
		return err
	}
	db, err := bolt.Open(dbfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		os.RemoveAll(dbfile)
		return err
	}
	boltDB := &BoltDB{db}
	defer boltDB.Close()

	// 创建保存数据使用的bucket
	for _, name := range DatabaseBuckets {
		if err := boltDB.createBucket(name); err != nil {
			return err
		}
	}
	// 创建默认作业队列
	if _, err := boltDB.putJSON("queue", constant.DefaultQueueName, model.JobQueueSpec{Name: "default", Enabled: true, Priority: 1000}); err != nil {
		return err
	}

	log.Println("Database file created")
	return nil
}

func (m *StateStore) loadFromDatabase() error {
	log.Printf("Loading server data from local database file \"%s\"...", m.dbPath)
	db, err := bolt.Open(m.dbPath, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return err
	}
	m.boltDB = &BoltDB{db}

	// 加载所有作业队列信息
	if err := m.boltDB.getBucketJSON("queue", func() interface{} {
		return &model.JobQueue{}
	}, func(v interface{}) {
		if queue, ok := v.(*model.JobQueue); ok {
			m.jobQueues[queue.Name] = queue
		}
	}); err != nil {
		return err
	}
	log.Printf("%d job queue(s) loaded", len(m.jobQueues))

	// 加载所有Job信息
	if err := m.boltDB.getBucketJSON("job", func() interface{} {
		return &model.Job{}
	}, func(v interface{}) {
		if job, ok := v.(*model.Job); ok {
			m.jobMap[job.ID] = job
		}
	}); err != nil {
		return err
	}
	// 排序所有Job
	m.jobList = make([]*model.Job, 0, len(m.jobMap))
	for _, v := range m.jobMap {
		m.jobList = append(m.jobList, v)
	}
	sort.Sort(model.GeneralJobSlice(m.jobList))
	log.Printf("%d job(s) loaded", len(m.jobList))

	// 加载所有Task信息
	if err := m.boltDB.getBucketJSON("task", func() interface{} {
		return &model.Task{}
	}, func(v interface{}) {
		if task, ok := v.(*model.Task); ok {
			j, g, t := model.ParseTaskID(task.ID)
			if job, ok := m.jobMap[j]; ok {
				job.Groups[g].Tasks[t] = task
			}
		}
	}); err != nil {
		return err
	}

	log.Printf("Server data loaded")
	return err
}

func (m *StateStore) GetJobQueue(name string) *model.JobQueue {
	queue, ok := m.jobQueues[name]
	if ok {
		return queue
	}
	return nil
}

func (m *StateStore) GetSchedulableQueues() []*model.JobQueue {
	queues := make([]*model.JobQueue, 0, len(m.jobQueues))
	for _, v := range m.jobQueues {
		if v.Enabled {
			queues = append(queues, v)
		}
	}
	if len(queues) <= 1 {
		return queues
	}
	sort.Sort(model.JobQueueSlice(queues))
	return queues
}

func (m *StateStore) GetJob(id string) *model.Job {
	job, ok := m.jobMap[id]
	if ok {
		return job
	}
	return nil
}

func (m *StateStore) AddJob(job *model.Job) error {
	// 确定ID的唯一性
	_, ok := m.jobMap[job.ID]
	if ok {
		return fmt.Errorf("Job ID \"%s\" conflicted with others", job.ID)
	}

	// 确定所属作业队列
	queue := m.GetJobQueue(job.Queue)
	if queue == nil {
		return fmt.Errorf("Invalid queue name \"%s\"", job.Queue)
	}
	job.SubmitTime = time.Now()

	// 写入数据库文件
	err := m.boltDB.put("job", job.ID, job.GetJSON())
	if err != nil {
		return fmt.Errorf("Unable to save submitted job \"%s\"(%s): %v", job.Name, job.ID, err)
	}

	// 追加到Job列表中
	queue.Jobs = append(queue.Jobs, job)
	m.jobMap[job.ID] = job
	m.jobList = append(m.jobList, job)

	log.Printf("Job \"%s\"(%s) with %d task(s) has beed added to queue \"%s\"", job.Name, job.ID, job.CountTasks(), job.Queue)
	return nil
}

func (m *StateStore) UpdateJobState(jobid string) error {
	job := m.GetJob(jobid)
	if job == nil {
		return fmt.Errorf("Job identified by \"%s\" not found", jobid)
	}
	if job.RefreshState() {
		log.Printf("  Job %s is set to \"%s\"\n", job.ID, model.JobStateString(job.State))
		err := m.boltDB.put("job", job.ID, job.GetJSON())
		if err != nil {
			return fmt.Errorf("Unable to save job \"%s\"(%s): %v", job.Name, job.ID, err)
		}
	}
	return nil
}

func (m *StateStore) SetJobState(jobid string, state model.JobState) error {
	job := m.GetJob(jobid)
	if job == nil {
		return fmt.Errorf("Job identified by \"%s\" not found", jobid)
	}
	job.State = state
	err := m.boltDB.put("job", job.ID, job.GetJSON())
	if err != nil {
		return fmt.Errorf("Unable to save job \"%s\"(%s): %v", job.Name, job.ID, err)
	}
	return nil
}

func (m *StateStore) SaveTasks(tasks []*model.Task) error {
	count := len(tasks)
	index := 0
	p := &index
	m.boltDB.putBatchJSON("task", func() (bool, string, interface{}) {
		eof := *p == (count - 1)
		*p = *p + 1
		return eof, tasks[*p-1].ID, tasks[*p-1]
	})
	return nil
}

func (m *StateStore) UpdateTaskStatus(id string, state model.TaskState, progress int, exit int, err string) *model.Task {
	jobid, gindex, tindex := model.ParseTaskID(id)
	if job, ok := m.jobMap[jobid]; !ok {
		log.Printf("No job identified by \"%s\" found while updating task status\n", jobid)
		return nil
	} else {
		task := job.Groups[gindex].Tasks[tindex]
		last := task.State
		task.State = state
		task.Progress = progress
		task.ExitCode = exit
		task.Error = err
		if task.State == model.TaskExecuting {
			task.StartTime = time.Now()
		} else if model.IsFinishState(state) {
			task.FinishTime = time.Now()
		}
		if last != task.State {
			// 仅在Task的状态发生变化时才保存
			m.boltDB.putJSON("task", task.ID, task)
			if model.IsFinishState(task.State) {
				log.Printf("  Task %s is reported as \"%s\" with exit code %d by node %s: %s", id, model.TaskStateString(task.State), exit, task.NodeName, err)
			} else {
				log.Printf("  Task %s is reported as \"%s\" by node %s", id, model.TaskStateString(task.State), task.NodeName)
			}
			// 当Task是执行状态并且Job也是执行状态时，无需更新Job的状态（此时可能仅仅是Task的进度刷新）
			if task.State != model.TaskExecuting || job.State != model.JobExecuting {
				if err := m.UpdateJobState(jobid); err != nil {
					log.Printf("Failed to update the state of job \"%s\": %v\n", jobid, err)
				}
			}
		}
		return task
	}
}
