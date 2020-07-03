package server

import (
	"fmt"

	"github.com/qianxiaoming/lightsched/common"
)

func (svc *APIServer) requestCreateJob(spec *common.JobSpec) error {
	svc.state.Lock()
	defer svc.state.Unlock()

	queue := svc.state.getJobQueue(spec.Queue)
	if queue == nil {
		return fmt.Errorf("Invalid queue name \"%s\"", spec.Queue)
	}
	return nil
}
