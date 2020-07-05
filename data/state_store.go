package data

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
	bolt "go.etcd.io/bbolt"
)

const (
	// DatabaseFileName 是系统默认主数据库的名字
	DatabaseFileName = "lightsched.db"
	// DefaultQueueName 是默认作业队列的名字
	DefaultQueueName = "default"
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
	m.dbPath = filepath.Join(util.GetCurrentPath(), path, DatabaseFileName)
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
	if err := boltDB.putJSON("queue", DefaultQueueName, model.JobQueueSpec{Name: "default", Enabled: true, Priority: 1000}); err != nil {
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

func (m *StateStore) GetJob(id string) *model.Job {
	job, ok := m.jobMap[id]
	if ok {
		return job
	}
	return nil
}

func (m *StateStore) AppendJob(job *model.Job) error {
	m.Lock()
	defer m.Unlock()

	// 确定所属作业队列
	queue := m.GetJobQueue(job.Queue)
	if queue == nil {
		return fmt.Errorf("Invalid queue name \"%s\"", job.Queue)
	}
	queue.Jobs = append(queue.Jobs, job)
	return nil
}
