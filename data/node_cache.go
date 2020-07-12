package data

import (
	"crypto/sha1"
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/message"
)

const (
	// NodeBucketCount 是保存节点消息的默认Bucket个数
	NodeBucketCount = 64
)

// NodeBucket 保存要发给节点的消息。多个节点可能会共享同一个NodeBucket。
type NodeBucket struct {
	sync.Mutex
	Messages map[string][]message.JSON
}

// NodeCache 记录了所有节点的信息，以及要分发给节点的Task信息
type NodeCache struct {
	sync.RWMutex
	nodeMap  map[string]*model.WorkNode
	nodeList []*model.WorkNode
	Buckets  [NodeBucketCount]NodeBucket
}

// NewNodeCache 创建新的节点缓存对象
func NewNodeCache() *NodeCache {
	cache := &NodeCache{
		nodeMap:  make(map[string]*model.WorkNode),
		nodeList: make([]*model.WorkNode, 0),
	}
	node1 := model.NewWorkNode("omen")
	node1.Address = "192.168.11.101"
	node1.Port = 20519
	node1.State = model.NodeOnline
	node1.Online = time.Now()
	node1.Resources.CPU.Cores = 24
	node1.Resources.CPU.Frequency = 24 * 3000
	node1.Resources.CPU.MinFreq = 3000
	node1.Resources.GPU.Cards = 4
	node1.Resources.GPU.Memory = 8
	node1.Resources.GPU.CUDA = 1020
	node1.Resources.Memory = 32000
	node1.Available = node1.Resources
	cache.AddNode(node1)

	node2 := model.NewWorkNode("scorpio")
	node2.Address = "192.168.11.102"
	node2.Port = 20519
	node2.State = model.NodeOnline
	node2.Online = time.Now()
	node2.Resources.CPU.Cores = 16
	node2.Resources.CPU.Frequency = 16 * 2400
	node2.Resources.CPU.MinFreq = 2400
	node2.Resources.GPU.Cards = 4
	node2.Resources.GPU.Memory = 11
	node2.Resources.GPU.CUDA = 1020
	node2.Resources.Memory = 32000
	node2.Available = node2.Resources
	cache.AddNode(node2)

	node3 := model.NewWorkNode("antares")
	node3.Address = "192.168.11.103"
	node3.Port = 20519
	node3.State = model.NodeOnline
	node3.Online = time.Now()
	node3.Resources.CPU.Cores = 16
	node3.Resources.CPU.Frequency = 16 * 3700
	node3.Resources.CPU.MinFreq = 3700
	node3.Resources.Memory = 64000
	node3.Available = node3.Resources
	cache.AddNode(node3)
	return cache
}

// GetNodes 获取当前记录的所有节点
func (cache *NodeCache) GetNodes() []*model.WorkNode {
	return cache.nodeList
}

// AddNode 加入新的节点
func (cache *NodeCache) AddNode(node *model.WorkNode) {
	cache.nodeMap[node.Address] = node
	cache.nodeList = append(cache.nodeList, node)

	index := int(sha1.Sum([]byte(node.Name))[0]) % NodeBucketCount
	cache.Buckets[index].Lock()
	defer cache.Buckets[index].Unlock()
	if cache.Buckets[index].Messages == nil {
		cache.Buckets[index].Messages = make(map[string][]message.JSON)
	}
	cache.Buckets[index].Messages[node.Name] = make([]message.JSON, 0, 8)
}

// AppendNodeMessage 给指定的节点增加一个消息
func (cache *NodeCache) AppendNodeMessage(name string, kind string, json []byte) {
	index := int(sha1.Sum([]byte(name))[0]) % NodeBucketCount
	cache.Buckets[index].Lock()
	defer cache.Buckets[index].Unlock()
	msgs := cache.Buckets[index].Messages[name]
	cache.Buckets[index].Messages[name] = append(msgs,
		message.JSON{
			Kind:    kind,
			Content: json,
		})
}

// TakeNodeMessage 获取指定节点的消息，然后清空它的消息列表
func (cache *NodeCache) TakeNodeMessage(name string) []message.JSON {
	index := int(sha1.Sum([]byte(name))[0]) % NodeBucketCount
	cache.Buckets[index].Lock()
	defer cache.Buckets[index].Unlock()
	msgs := cache.Buckets[index].Messages[name]
	cache.Buckets[index].Messages[name] = make([]message.JSON, 0, 8)
	return msgs
}
