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
	dbFile    *bolt.DB
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
	if m.dbFile != nil {
		m.dbFile.Close()
	}
	m.dbFile = nil
}

func createDatabaseFile(dbfile string) error {
	log.Printf("Creating database file \"%s\" on first startup...", dbfile)
	if err := util.MakeDirAll(filepath.Dir(dbfile)); err != nil {
		return err
	}
	dbFile, err := bolt.Open(dbfile, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		os.RemoveAll(dbfile)
		return err
	}
	defer dbFile.Close()
	log.Println("Database file created")
	return nil
}

func (m *StateModel) loadStateFromDatabase() error {
	dbFile, err := bolt.Open(m.dbPath, 0600, &bolt.Options{Timeout: 3 * time.Second})
	m.dbFile = dbFile
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
