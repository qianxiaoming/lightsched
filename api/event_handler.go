package api

import (
	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/common"
)

type RestEventHandler interface {
	OnCreateJobEvent(spec *common.JobSpec) error
	RestRouter() *gin.Engine
}

type NodeEventHandler interface {
	OnNodeEvent()
}

type TaskEventHandler interface {
	OnTaskEvent()
}
