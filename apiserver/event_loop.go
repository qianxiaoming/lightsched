package apiserver

import "github.com/gin-gonic/gin"

func (svc *APIServer) OnRestEvent() {

}

func (svc *APIServer) RestRouter() *gin.Engine {
	return svc.restRouter
}

func (svc *APIServer) OnNodeEvent() {

}

func (svc *APIServer) OnTaskEvent() {

}

func (svc *APIServer) EventLoop() {

}
