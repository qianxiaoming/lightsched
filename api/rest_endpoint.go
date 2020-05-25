package api

import "github.com/gin-gonic/gin"

type RestEndpoint interface {
	registerRoute()
	restPrefix() string
}

type JobEndpoint struct {
	handler RestEventHandler
}

func (job *JobEndpoint) registerRoute() {
	job.handler.RestRouter().GET(job.restPrefix(), func(c *gin.Context) {
		job.handler.OnRestEvent()
		c.JSON(200, gin.H{
			"jobs": [...]string{"001", "002", "003"},
		})
	})
}

func (job *JobEndpoint) restPrefix() string {
	return "/jobs"
}

type TaskEndpoint struct {
	handler RestEventHandler
}

func (task *TaskEndpoint) registerRoute() {
	task.handler.RestRouter().GET(task.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (task *TaskEndpoint) restPrefix() string {
	return "/tasks"
}

type QueueEndpoint struct {
	handler RestEventHandler
}

func (queue *QueueEndpoint) registerRoute() {
	queue.handler.RestRouter().GET(queue.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (queue *QueueEndpoint) restPrefix() string {
	return "/queues"
}

type NodeEndpoint struct {
	handler RestEventHandler
}

func (node *NodeEndpoint) registerRoute() {
	node.handler.RestRouter().GET(node.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (node *NodeEndpoint) restPrefix() string {
	return "/nodes"
}

type UserEndpoint struct {
	handler RestEventHandler
}

func (user *UserEndpoint) registerRoute() {
	user.handler.RestRouter().GET(user.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (user *UserEndpoint) restPrefix() string {
	return "/users"
}
