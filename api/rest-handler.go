package api

import "github.com/gin-gonic/gin"

type RestHandler interface {
	registerRoute()
	restPrefix() string
}

type JobRestHandler struct {
	svc *APIServer
}

func (h *JobRestHandler) registerRoute() {
	h.svc.restRouter.GET(h.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"jobs": [...]string{"001", "002", "003"},
		})
	})
}

func (h *JobRestHandler) restPrefix() string {
	return "/jobs"
}

type TaskRestHandler struct {
	svc *APIServer
}

func (h *TaskRestHandler) registerRoute() {
	h.svc.restRouter.GET(h.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (h *TaskRestHandler) restPrefix() string {
	return "/tasks"
}

type QueueRestHandler struct {
	svc *APIServer
}

func (h *QueueRestHandler) registerRoute() {
	h.svc.restRouter.GET(h.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (h *QueueRestHandler) restPrefix() string {
	return "/queues"
}

type NodeRestHandler struct {
	svc *APIServer
}

func (h *NodeRestHandler) registerRoute() {
	h.svc.restRouter.GET(h.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (h *NodeRestHandler) restPrefix() string {
	return "/nodes"
}

type UserRestHandler struct {
	svc *APIServer
}

func (h *UserRestHandler) registerRoute() {
	h.svc.restRouter.GET(h.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (h *UserRestHandler) restPrefix() string {
	return "/users"
}
