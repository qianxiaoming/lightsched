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
		// TODO 这里应该加入对API Server的回应：Task执行失败
		log.Printf("Unable to unmarshal task json: %v\n", err)
	} else {
		log.Printf("Execute task(%s) program: %s %s\n", task.ID, task.Command, task.Args)
		cmd := exec.Command(task.Command, task.Args)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		// for linux TODO
		// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, Setpgid: true}

		if len(task.WorkDir) > 0 {
			cmd.Dir = task.WorkDir
		}
		if len(task.Envs) > 0 {
			cmd.Env = task.Envs
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Cannot get standard output pipe: %v\n", err)
			// TODO
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			log.Printf("Cannot start program for task(%s): %v\n", task.ID, err)
			// TODO 发送消息给主线程，通知任务失败
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
				log.Printf("Program exit error: %v\n", exit)
			}
		} else {
			log.Println("Program exit successfully")
		}
	}
}
