package data

import (
	"sync"
	"time"

	"github.com/qianxiaoming/lightsched/model"
)

// NodeCache 记录了所有节点的信息，以及要分发给节点的Task信息
type NodeCache struct {
	sync.RWMutex
	nodeMap  map[string]*model.WorkNode
	nodeList []*model.WorkNode
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
	node1.Resources.GPU.Cards = 4
	node1.Resources.GPU.Cores = 2048
	node1.Resources.GPU.Memory = 11000
	node1.Resources.GPU.CUDA = 1020
	node1.Resources.Memory = 32000
	cache.nodeMap[node1.Address] = node1
	cache.nodeList = append(cache.nodeList, node1)

	node2 := model.NewWorkNode("scorpio")
	node2.Address = "192.168.11.102"
	node2.Port = 20519
	node2.State = model.NodeOnline
	node2.Online = time.Now()
	node2.Resources.CPU.Cores = 24
	node2.Resources.CPU.Frequency = 24 * 3000
	node2.Resources.GPU.Cards = 4
	node2.Resources.GPU.Cores = 2048
	node2.Resources.GPU.Memory = 11000
	node2.Resources.GPU.CUDA = 1020
	node2.Resources.Memory = 32000
	cache.nodeMap[node2.Address] = node2
	cache.nodeList = append(cache.nodeList, node2)
	return cache
}

// GetNodes 获取当前记录的所有节点
func (cache *NodeCache) GetNodes() []*model.WorkNode {
	return cache.nodeList
}
