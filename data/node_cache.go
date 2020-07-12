package data

import (
	"crypto/sha1"
	"log"
	"sync"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
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
	return cache
}

// GetNodes 获取当前记录的所有节点
func (cache *NodeCache) GetNodes() []*model.WorkNode {
	return cache.nodeList
}

// AddNode 加入新的节点
func (cache *NodeCache) AddNode(node *model.WorkNode) {
	if v, ok := cache.nodeMap[node.Name]; ok {
		log.Printf("Node named \"%s\" already exists and its state is %d", v.Name, v.State)
	}
	cache.nodeMap[node.Name] = node
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
