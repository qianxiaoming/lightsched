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
		log.Printf("Unable to unmarshal task json: %v\n", err)
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
			node.notifyTaskStatus(task.ID, model.TaskAborted, 0, 0, err.Error())
			return
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			log.Printf("Cannot start program for task(%s): %v\n", task.ID, err)
			node.notifyTaskStatus(task.ID, model.TaskAborted, 0, 0, err.Error())
			return
		}

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
				log.Printf("Task(%s) program exit error: %v\n", task.ID, err)
				if exit.Success() {
					log.Printf("Task(%s) program exit successfully\n", task.ID)
					node.notifyTaskStatus(task.ID, model.TaskCompleted, 100, 0, "")
				} else {
					node.notifyTaskStatus(task.ID, model.TaskFailed, -1, exit.ExitCode(), exit.Error())
				}
			}
		} else {
			log.Printf("Task(%s) program exit successfully\n", task.ID)
			node.notifyTaskStatus(task.ID, model.TaskCompleted, 100, 0, "")
		}
	}
}
