package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
)

// NodeRegisterEndpoint 是计算节点向主节点注册的接口Node
type NodeRegisterEndpoint struct{}

func (e NodeRegisterEndpoint) registerRoute() {
	apiserver.nodeRouter.POST(e.restPrefix(), func(c *gin.Context) {
		reg := &message.RegisterNode{}
		if err := c.BindJSON(reg); err == nil {
			ip := c.ClientIP()
			log.Printf("Node \"%s\" try to register...\n", reg.Name)
			log.Printf("    Address:  %s", ip)
			log.Printf("    Platform: %s (%s %s %s)", reg.Platform.Kind, reg.Platform.Name, reg.Platform.Family, reg.Platform.Version)
			log.Printf("    CPU Info: %d cores %dMHz", int(reg.Resources.CPU.Cores), reg.Resources.CPU.MinFreq)
			log.Printf("    Mem Info: %dGi", reg.Resources.Memory/1024)
			log.Printf("    GPU Info: %d card(s) %dGi with CUDA %.1f", int(reg.Resources.GPU.Cards), reg.Resources.GPU.Memory, float32(reg.Resources.GPU.CUDA)/100.0)
			err = apiserver.requestRegisterNode(ip, reg)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"cluster": apiserver.config.Cluster})
				log.Println("Node registered")
			} else {
				responseError(http.StatusNotAcceptable, "%v", err, c)
			}
		} else {
			responseError(http.StatusBadRequest, "Parse request failed: %v", err, c)
		}
	})
}

func (e NodeRegisterEndpoint) restPrefix() string {
	return "/nodes"
}

// HeartbeatEndpoint 是计算节点向主节点发送心跳信息的接口
type HeartbeatEndpoint struct{}

func (e HeartbeatEndpoint) registerRoute() {
	apiserver.nodeRouter.POST(e.restPrefix(), func(c *gin.Context) {
		hb := &message.Heartbeat{}
		if err := c.BindJSON(hb); err == nil {
			msgs, found := apiserver.nodes.PeriodicUpdate(hb.Name, hb.CPU, hb.Memory, hb.Executings)
			status := http.StatusOK
			if !found {
				status = http.StatusNotFound
			}
			if msgs == nil {
				c.Status(status)
			} else {
				c.JSON(status, msgs)
			}
			// 更新上报的Task状态
			if len(hb.Payload) != 0 {
				go apiserver.requestUpdateTasks(hb.Payload)
			}
		} else {
			log.Printf("Invalid heartbeat from %s: %v\n", c.ClientIP(), err)
			responseError(http.StatusBadRequest, "Parse request failed: %v", err, c)
		}
	})
}

func (e HeartbeatEndpoint) restPrefix() string {
	return "/heartbeat"
}

// TaskLogEndpoint 是计算节点向主节点发送Task日志的接口
type TaskLogEndpoint struct{}

func (e TaskLogEndpoint) registerRoute() {
	apiserver.nodeRouter.POST(e.restPrefix(), func(c *gin.Context) {
		jobid, gindex, tindex := model.ParseTaskID(c.Param("taskid"))
		filename := filepath.Join(apiserver.config.DataPath, jobid, fmt.Sprintf("%d.%d.log", gindex, tindex))
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("Unable to create log file %s: %v\n", c.Param("taskid"), err)
			responseError(http.StatusInternalServerError, "Failed to save task log: %v", err, c)
		}
		defer file.Close()
		if _, err := io.Copy(file, c.Request.Body); err == nil {
			c.Status(http.StatusOK)
		} else {
			responseError(http.StatusInternalServerError, "Failed to save task log: %v", err, c)
		}
	})
}

func (e TaskLogEndpoint) restPrefix() string {
	return "/tasks/:taskid/log"
}
