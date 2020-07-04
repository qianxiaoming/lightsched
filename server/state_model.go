package server

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/common"
	"github.com/qianxiaoming/lightsched/util"
	bolt "go.etcd.io/bbolt"
)

const (
	DatabaseFileName = "lightsched.db"
)

// StateModel 是API Server的内部状态数据
type StateModel struct {
	sync.RWMutex
	dbPath    string
	boltDB    *BoltDB
	jobQueues map[string]*common.JobQueue
	jobMap    map[string]*common.Job
	jobList   []*common.Job
}

// NewStateModel 创建服务的内部状态数据对象
func NewStateModel() *StateModel {
	return &StateModel{
		jobQueues: make(map[string]*common.JobQueue),
		jobMap:    make(map[string]*common.Job),
		jobList:   make([]*common.Job, 0, 128),
	}
}

func (m *StateModel) initState(path string) error {
	m.dbPath = filepath.Join(util.GetCurrentPath(), path, DatabaseFileName)
	if !util.PathExists(m.dbPath) {
		if err := createDatabaseFile(m.dbPath); err != nil {
			return err
		}
	}
	if err := m.loadStateFromDatabase(); err != nil {
		return err
	}
	return nil
}

func (m *StateModel) clearState() {
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
	buckets := []string{"config", "queue", "job", "task"}
	for _, name := range buckets {
		if err := boltDB.createBucket(name); err != nil {
			return err
		}
	}
	// 创建默认作业队列
	if err := boltDB.putJSON("queue", "default", common.NewJobQueue("default", true, 1000)); err != nil {
		return err
	}

	log.Println("Database file created")
	return nil
}

func (m *StateModel) loadStateFromDatabase() error {
	log.Printf("Loading server data from local database file \"%s\"...", m.dbPath)
	db, err := bolt.Open(m.dbPath, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return err
	}
	m.boltDB = &BoltDB{db}

	// 加载所有作业队列信息
	if err := m.boltDB.getBucketJSON("queue", func() interface{} {
		return &common.JobQueue{}
	}, func(v interface{}) {
		if queue, ok := v.(*common.JobQueue); ok {
			m.jobQueues[queue.Name] = queue
		}
	}); err != nil {
		return err
	}
	log.Printf("%d job queue(s) loaded", len(m.jobQueues))

	log.Printf("Server data loaded")
	return err
}

func (m *StateModel) getJobQueue(name string) *common.JobQueue {
	queue, ok := m.jobQueues[name]
	if ok {
		return queue
	}
	return nil
}

func (m *StateModel) getJob(id string) *common.Job {
	job, ok := m.jobMap[id]
	if ok {
		return job
	}
	return nil
}
