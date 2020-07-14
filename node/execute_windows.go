package node

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os/exec"
	"syscall"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
)

func (node *NodeServer) runExecuteTask(msg message.JSON) {
	task := &model.Task{}
	if err := json.Unmarshal(msg.Content, task); err != nil {
		log.Printf("NOTICE: Unable to unmarshal task json and just ignore it now: %v\n", err)
	} else {
		log.Printf("Execute task(%s) program: %s %s\n", task.ID, task.Command, task.Args)
		cmd := exec.Command(task.Command, task.Args)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		if len(task.WorkDir) > 0 {
			cmd.Dir = task.WorkDir
		}
		if len(task.Envs) > 0 {
			cmd.Env = task.Envs
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Cannot get standard output pipe for task(%s): %v\n", task.ID, err)
			node.notifyTaskStatus(task.ID, model.TaskAborted, 0, 0, 0, err.Error())
			return
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			log.Printf("Cannot start program for task(%s): %v\n", task.ID, err)
			node.notifyTaskStatus(task.ID, model.TaskAborted, 0, 0, 0, err.Error())
			return
		}
		node.notifyTaskStatus(task.ID, model.TaskExecuting, cmd.Process.Pid, 0, 0, "")

		progress := 0
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}
			// TODO
			log.Println(">>>>", line)
		}
		if err := cmd.Wait(); err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				if exit.Success() {
					log.Printf("Task(%s) program exit successfully\n", task.ID)
					node.notifyTaskStatus(task.ID, model.TaskCompleted, 0, progress, 0, "")
				} else {
					log.Printf("Task(%s) program exit error: %d %v\n", task.ID, exit.ExitCode(), err)
					node.notifyTaskStatus(task.ID, model.TaskFailed, 0, progress, exit.ExitCode(), exit.Error())
				}
			}
		} else {
			log.Printf("Task(%s) program exit successfully\n", task.ID)
			node.notifyTaskStatus(task.ID, model.TaskCompleted, 0, progress, 0, "")
		}
	}
}
