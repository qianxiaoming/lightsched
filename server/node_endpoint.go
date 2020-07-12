package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/message"
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

// NodeRegisterEndpoint 是计算节点向主节点注册的接口Node
type NodeRegisterEndpoint struct {
	server *APIServer
}

func (e *NodeRegisterEndpoint) registerRoute() {
	e.server.nodeRouter.POST(e.restPrefix(), func(c *gin.Context) {
		reg := &message.RegisterNode{}
		if err := c.BindJSON(reg); err == nil {
			ip := c.ClientIP()
			log.Printf("Node \"%s\" try to register...\n", reg.Name)
			log.Printf("    Address:  %s", ip)
			log.Printf("    Platform: %s (%s %s %s)", reg.Platform.Kind, reg.Platform.Name, reg.Platform.Family, reg.Platform.Version)
			log.Printf("    CPU Info: %d cores %dMHz", int(reg.Resources.CPU.Cores), reg.Resources.CPU.MinFreq)
			log.Printf("    Mem Info: %dGi", reg.Resources.Memory/1024)
			log.Printf("    GPU Info: %d card(s) %dGi with CUDA %.1f", int(reg.Resources.GPU.Cards), reg.Resources.GPU.Memory, float32(reg.Resources.GPU.CUDA)/100.0)
			err = e.server.requestRegisterNode(ip, reg)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"ack": "ok"})
				log.Println("Node registered")
			} else {
				responseError(http.StatusNotAcceptable, "%v", err, c)
			}
		} else {
			responseError(http.StatusBadRequest, "Parse request failed: %v", err, c)
		}
	})
}

func (e *NodeRegisterEndpoint) restPrefix() string {
	return "/nodes"
}
