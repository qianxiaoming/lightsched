package api

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/common"
)

type EventWrapper struct {
	Params   interface{}
	Response chan interface{}
}

func (svc *APIServer) OnCreateJobEvent(spec *common.JobSpec) error {
	// 构造Web请求事件并传入事件循环中
	ch := make(chan interface{})
	svc.restChan <- &EventWrapper{spec, ch}
	err := <-ch
	close(ch)
	if err == nil {
		return nil
	}
	return err.(error)
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
		//event * EventWrapper
		select {
		case e := <-svc.restChan:
			log.Println("RESTful Event received")
			event := e.(*EventWrapper)
			if spec, ok := event.Params.(*common.JobSpec); ok {
				log.Printf("Begin to create job %s(%s)...", spec.Name, spec.ID)
				event.Response <- nil
			} else {
				event.Response <- fmt.Errorf("Wrong job spec type")
			}
		case <-svc.nodeChan:
		case <-svc.taskChan:
		case <-svc.stopChan:
			log.Println("Event loop of API Server stopped")
			return
		}
	}
}
