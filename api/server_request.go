package api

import (
	"fmt"

	"github.com/qianxiaoming/lightsched/common"
)

func (svc *APIServer) requestCreateJob(spec *common.JobSpec) error {
	svc.Lock()
	defer svc.Unlock()

	queue := svc.state.getJobQueue(spec.Queue)
	if queue == nil {
		return fmt.Errorf("Invalid queue name \"%s\"", spec.Queue)
	}
	return nil
}
