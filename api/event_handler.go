package apiserver

import "github.com/gin-gonic/gin"

type RestEventHandler interface {
	OnRestEvent()
	RestRouter() *gin.Engine
}

type NodeEventHandler interface {
	OnNodeEvent()
}

type TaskEventHandler interface {
	OnTaskEvent()
}
