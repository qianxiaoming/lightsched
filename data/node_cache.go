package data

import (
	"crypto/sha1"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
)

const (
	// NodeBucketCount 是保存节点消息的默认Bucket个数
	NodeBucketCount = 64
)

// NodePeriodic 是节点周期性更新的数据
type NodePeriodic struct {
	timestamp time.Time
	cpu       float64
	mem       float64
}

// NodeBucket 保存要发给节点的消息。多个节点可能会共享同一个NodeBucket。
type NodeBucket struct {
	sync.Mutex
	messages  map[string][]*message.JSON
	periodics map[string]*NodePeriodic
}

// NodeCache 记录了所有节点的信息，以及要分发给节点的Task信息
type NodeCache struct {
	sync.RWMutex
	nodeMap  map[string]*model.WorkNode
	nodeList []*model.WorkNode
	buckets  [NodeBucketCount]NodeBucket
}

// NewNodeCache 创建新的节点缓存对象
func NewNodeCache() *NodeCache {
	cache := &NodeCache{
		nodeMap:  make(map[string]*model.WorkNode),
		nodeList: make([]*model.WorkNode, 0),
	}
	return cache
}

// GetNodes 获取当前记录的所有节点
func (cache *NodeCache) GetNodes() []*model.WorkNode {
	return cache.nodeList
}

// GetNode 返回指定名字的节点
func (cache *NodeCache) GetNode(name string) *model.WorkNode {
	if node, ok := cache.nodeMap[name]; ok {
		return node
	}
	return nil
}

// AddNode 加入新的节点
func (cache *NodeCache) AddNode(node *model.WorkNode) {
	if v, ok := cache.nodeMap[node.Name]; ok {
		log.Printf("Node named \"%s\" already exists and its state is %d", v.Name, v.State)
	}
	cache.nodeMap[node.Name] = node
	cache.nodeList = append(cache.nodeList, node)

	index := int(sha1.Sum([]byte(node.Name))[0]) % NodeBucketCount
	cache.buckets[index].Lock()
	defer cache.buckets[index].Unlock()
	if cache.buckets[index].messages == nil {
		cache.buckets[index].messages = make(map[string][]*message.JSON)
		cache.buckets[index].periodics = make(map[string]*NodePeriodic)
	}
	cache.buckets[index].messages[node.Name] = nil
}

// AppendNodeMessage 给指定的节点增加一个消息
func (cache *NodeCache) AppendNodeMessage(name string, kind string, object string, content []byte) {
	index := int(sha1.Sum([]byte(name))[0]) % NodeBucketCount
	cache.buckets[index].Lock()
	defer cache.buckets[index].Unlock()
	msgs := cache.buckets[index].messages[name]
	if msgs == nil {
		msgs = make([]*message.JSON, 0, 8)
	}
	msg := &message.JSON{
		Kind:    kind,
		Object:  object,
		Content: content,
	}
	// 需要根据新消息过滤同一个节点上的其它消息
	filterMsg := false
	for _, old := range msgs {
		if !message.Filter(msg, old) {
			filterMsg = true
			break
		}
	}
	if filterMsg {
		news := make([]*message.JSON, 0, len(msgs))
		for _, old := range msgs {
			if message.Filter(msg, old) {
				news = append(news, old)
			} else {
				log.Printf("Message of kind %s for %s will not send to %s\n", old.Kind, old.Object, name)
				if old.Kind == message.KindScheduleTask {
					// 需要将资源归还给节点
					task := &model.Task{}
					if err := json.Unmarshal(old.Content, task); err == nil {
						log.Println("Give back resources of this task")
						cache.nodeMap[task.NodeName].Available.GiveBack(task.Resources)
					}
				}
			}
		}
		msgs = news
	}
	cache.buckets[index].messages[name] = append(msgs, msg)
}

// PeriodicUpdate 获取指定节点的消息，然后清空它的消息列表。返回false表示未发现该节点的注册信息。
func (cache *NodeCache) PeriodicUpdate(name string, cpu float64, mem float64) ([]*message.JSON, bool) {
	index := int(sha1.Sum([]byte(name))[0]) % NodeBucketCount
	cache.buckets[index].Lock()
	defer cache.buckets[index].Unlock()
	// 更新节点的时间戳及状态信息
	if cache.buckets[index].periodics == nil {
		return nil, false
	}
	if update, ok := cache.buckets[index].periodics[name]; !ok {
		update = &NodePeriodic{
			timestamp: time.Now(),
			cpu:       cpu,
			mem:       mem,
		}
		cache.buckets[index].periodics[name] = update
		log.Printf("Heartbeat for node \"%s\" received for the first time: CPU usage = %.2f, Memory usage = %.2f", name, cpu, mem)
	} else {
		update.timestamp = time.Now()
		update.cpu = cpu
		update.mem = mem
	}
	// 获取要发送给节点的消息
	if msgs, ok := cache.buckets[index].messages[name]; ok {
		cache.buckets[index].messages[name] = nil
		return msgs, true
	}
	return nil, true
}
