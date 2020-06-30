package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/common"
)

// RestEndpoint 是对不同资源对象提供RESTful API实现的接口
type RestEndpoint interface {
	registerRoute()
	restPrefix() string
}

func responseError(code int, format string, err error, c *gin.Context) {
	str := fmt.Sprintf(format, err)
	log.Printf(str)
	if code != 0 {
		c.String(code, str)
	} else {
		c.Writer.WriteString(str)
	}
}

// JobEndpoint 是Job资源对象的RESTful API实现接口
type JobEndpoint struct {
	handler *APIServer
}

func (job *JobEndpoint) registerRoute() {
	job.handler.restRouter.GET(job.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"jobs": [...]string{"001", "002", "003"},
		})
	})
	job.handler.restRouter.POST(job.restPrefix(), job.createJob)
	job.handler.restRouter.DELETE(job.restPrefix()+"/:id", job.deleteJob)
}

func (job *JobEndpoint) restPrefix() string {
	return "/jobs"
}

func (job *JobEndpoint) createJob(c *gin.Context) {
	var spec common.JobSpec
	if err := c.BindJSON(&spec); err == nil {
		log.Printf("Request to create job \"%s\"(%s) in queue \"%s\" with %d task group(s)...\n", spec.Name, spec.ID, spec.Queue, len(spec.Groups))
		err = job.handler.requestCreateJob(&spec)
		if err == nil {
			c.JSON(http.StatusCreated, gin.H{"id": spec.ID})
		} else {
			responseError(http.StatusBadRequest, "Create job failed: %v", err, c)
		}
	} else {
		responseError(http.StatusBadRequest, "Parse request failed: %v", err, c)
	}
}

func (job *JobEndpoint) deleteJob(c *gin.Context) {
	id := c.Params.ByName("id")
	c.JSON(http.StatusOK, gin.H{"job": id})
}

// TaskEndpoint 是Task资源对象的RESTful API实现接口
type TaskEndpoint struct {
	handler *APIServer
}

func (task *TaskEndpoint) registerRoute() {
	task.handler.restRouter.GET(task.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (task *TaskEndpoint) restPrefix() string {
	return "/tasks"
}

// QueueEndpoint 是Queue资源对象的RESTful API实现接口
type QueueEndpoint struct {
	handler *APIServer
}

func (queue *QueueEndpoint) registerRoute() {
	queue.handler.restRouter.GET(queue.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (queue *QueueEndpoint) restPrefix() string {
	return "/queues"
}

// NodeEndpoint 是Node资源对象的RESTful API实现接口
type NodeEndpoint struct {
	handler *APIServer
}

func (node *NodeEndpoint) registerRoute() {
	node.handler.restRouter.GET(node.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (node *NodeEndpoint) restPrefix() string {
	return "/nodes"
}

// UserEndpoint 是User资源对象的RESTful API实现接口
type UserEndpoint struct {
	handler *APIServer
}

func (user *UserEndpoint) registerRoute() {
	user.handler.restRouter.GET(user.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (user *UserEndpoint) restPrefix() string {
	return "/users"
}
