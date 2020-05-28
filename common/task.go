package common

type Task struct {
}

// TaskGroup 表示一组执行命令相同的任务，但每个任务的参数可以不同
type TaskGroup struct {
	Envs      []string
	Command   string
	WorkDir   string
	PreGroups []*TaskGroup
	tasks     []*Task
}
