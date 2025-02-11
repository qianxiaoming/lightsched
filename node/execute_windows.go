package node

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/qianxiaoming/lightsched/model"
	"github.com/qianxiaoming/lightsched/util"
)

func (node *NodeServer) runExecuteTask(msg *message.JSON) {
	task := &model.Task{}
	if err := json.Unmarshal(msg.Content, task); err != nil {
		log.Printf("NOTICE: Unable to unmarshal task json and just ignore it now: %v\n", err)
	} else {
		command := task.Command
		workdir := task.WorkDir
		if !util.PathExists(command) {
			command = filepath.Join(filepath.Dir(util.GetCurrentPath()), command)
			if !util.PathExists(command) {
				log.Printf("Cannot find program for task(%s): %s does not exist\n", task.ID, task.Command)
				node.notifyTaskStatus(task.ID, model.TaskAborted, nil, 0, 0, "Program does not exist")
				return
			}
			workdir = filepath.Dir(command)
		}
		log.Printf("Execute task(%s) program: %s %s\n", task.ID, command, task.Args)
		cmd := exec.Command(command, strings.Split(task.Args, " ")...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		if len(workdir) > 0 {
			cmd.Dir = workdir
		}
		if len(task.Envs) > 0 {
			cmd.Env = task.Envs
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Cannot get standard output pipe for task(%s): %v\n", task.ID, err)
			node.notifyTaskStatus(task.ID, model.TaskAborted, nil, 0, 0, err.Error())
			return
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			log.Printf("Cannot start program for task(%s): %v\n", task.ID, err)
			node.notifyTaskStatus(task.ID, model.TaskAborted, nil, 0, 0, err.Error())
			return
		}
		node.notifyTaskStatus(task.ID, model.TaskExecuting, cmd.Process, 0, 0, "")

		var logs strings.Builder
		progress := 0
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}
			// 记录任务程序的输出
			if strings.HasPrefix(line, "[PROGRESS]") {
				cur, str := parseProgress(line)
				if cur != -1 && cur != progress {
					progress = cur
					node.notifyTaskStatus(task.ID, model.TaskExecuting, cmd.Process, progress, 0, "")
				}
				if len(str) > 0 {
					logs.WriteString(str)
				}
			} else if strings.HasPrefix(line, "[ERROR]") {
				s := strings.Index(line, "]")
				if s < len(line)-1 {
					line = strings.Trim(line[s+1:], " ")
					logs.WriteString(line)
					line = strings.Trim(line, "\r\n")
					node.notifyTaskStatus(task.ID, model.TaskExecuting, cmd.Process, progress, 0, line)
				}
			} else {
				logs.WriteString(line)
			}
		}
		if err := cmd.Wait(); err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				if exit.Success() {
					log.Printf("Task(%s) program exit successfully\n", task.ID)
					node.notifyTaskStatus(task.ID, model.TaskCompleted, nil, progress, 0, "")
				} else {
					log.Printf("Task(%s) program exit error: %v\n", task.ID, err)
					node.notifyTaskStatus(task.ID, model.TaskFailed, nil, progress, exit.ExitCode(), exit.Error())
				}
			}
		} else {
			log.Printf("Task(%s) program exit successfully\n", task.ID)
			node.notifyTaskStatus(task.ID, model.TaskCompleted, nil, progress, 0, "")
		}
		// 将日志发送给API Server
		if logs.Len() > 0 && node.state != model.NodeUnknown {
			url := fmt.Sprintf(node.config.LogURL, task.ID)
			if _, err := http.Post(url, "text/plain", bytes.NewReader([]byte(logs.String()))); err != nil {
				log.Printf("Unable to post logs for task %s: %v\n", task.ID, err)
			}
		}
	}
}

func parseProgress(str string) (int, string) {
	s := strings.Index(str, " ")
	e := strings.Index(str, "%")
	if s != -1 && e != -1 {
		if p, err := strconv.Atoi(str[s+1 : e]); err == nil {
			if p > 100 {
				p = 100
			}
			if e < len(str)-2 {
				return p, str[e+2:]
			}
			return p, ""
		}
	}
	return -1, ""
}
