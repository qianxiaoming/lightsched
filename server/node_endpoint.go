package server

import (
	"github.com/gin-gonic/gin"
)

// HeartbeatEndpoint 是计算节点向主节点发送心跳信息的接口
type HeartbeatEndpoint struct {
	server *APIServer
}

func (e *HeartbeatEndpoint) registerRoute() {
	e.server.nodeRouter.POST(e.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"ack": "ok",
		})
	})
}

func (e *HeartbeatEndpoint) restPrefix() string {
	return "/heartbeat"
}
