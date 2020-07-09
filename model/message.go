package model

const (
	// MsgScheduleTask 调度任务执行消息
	MsgScheduleTask = "task"
)

// JSONMessage 表示内容是JSON的消息，其中通过kind字段说明类型
type JSONMessage struct {
	Kind string
	JSON []byte
}
