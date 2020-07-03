package api

import (
	"github.com/gin-gonic/gin"
)

// HeartbeatEndpoint 是计算节点向主节点发送心跳信息的接口
type HeartbeatEndpoint struct {
	handler *APIServer
}

func (hb *HeartbeatEndpoint) registerRoute() {
	hb.handler.nodeRouter.POST(hb.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"ack": "ok",
		})
	})
}

func (hb *HeartbeatEndpoint) restPrefix() string {
	return "/heartbeat"
}
