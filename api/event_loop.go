package apiserver

import (
	"log"

	"github.com/gin-gonic/gin"
)

func (svc *APIServer) OnRestEvent() {
	svc.restChan <- struct{}{}
}

func (svc *APIServer) RestRouter() *gin.Engine {
	return svc.restRouter
}

func (svc *APIServer) OnNodeEvent() {

}

func (svc *APIServer) OnTaskEvent() {

}

// EventLoop 是API Server的主事件循环实现
func (svc *APIServer) EventLoop() {
	for {
		select {
		case <-svc.restChan:
			log.Println("RESTful Event received")
		case <-svc.nodeChan:
		case <-svc.taskChan:
		case <-svc.stopChan:
			log.Println("Event loop of API Server stopped")
			return
		}
	}
}
