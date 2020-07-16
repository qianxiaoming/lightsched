package node

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

func (node *NodeServer) sendHeartbeat() error {
	cpu, _ := cpu.Percent(0, false)
	mem, _ := mem.VirtualMemory()
	var payload []*message.TaskStatus
	if len(node.heartbeat.payload) > 0 {
		payload = make([]*message.TaskStatus, 0, len(node.heartbeat.payload))
		for _, v := range node.heartbeat.payload {
			payload = append(payload, v)
		}
		node.heartbeat.payload = make(map[string]*message.TaskStatus)
	}
	hb := &message.Heartbeat{
		Name:    node.config.hostname,
		CPU:     cpu[0],
		Memory:  mem.UsedPercent,
		Payload: payload,
	}
	request, _ := json.Marshal(hb)
	if resp, err := http.Post(node.heartbeat.url, "application/json", bytes.NewReader(request)); err != nil {
		log.Printf("Send heartbeat failed: %v\n", err)
		// 心跳发送失败时需要恢复原来的待发送信息
		for _, status := range payload {
			node.heartbeat.payload[status.ID] = status
		}
		return err
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) == 0 {
			return nil
		}
		msgs := make([]message.JSON, 0)
		if err = json.Unmarshal(body, &msgs); err != nil {
			log.Printf("Unable to unmarshal the response for heartbeat: %v", err)
		} else {
			node.runServerMessages(msgs)
		}
	}
	return nil
}

func (node *NodeServer) runServerMessages(msgs []message.JSON) {
	// 确定此次消息中包含的终止Job的消息
	var terminated map[string]bool
	for _, msg := range msgs {
		if msg.Kind != message.KindTerminateJob {
			continue
		}
		jobid := &message.JobID{}
		if err := json.Unmarshal(msg.Content, jobid); err != nil {
			log.Printf("NOTICE: Unable to unmarshal jobid and just ignore it now: %v\n", err)
			continue
		}
		if terminated == nil {
			terminated = make(map[string]bool)
		}
		terminated[jobid.ID] = true
	}

	for _, msg := range msgs {
		switch msg.Kind {
		case message.KindScheduleTask:
			go node.runExecuteTask(msg, terminated)
		}
	}

	for jobid := range terminated {
		log.Printf("Terminating job %s...\n", jobid)
		for id, proc := range node.executings {
			if strings.HasPrefix(id, jobid) && proc.process != nil {
				log.Printf("  Killing task(%s) process %d...\n", id, proc.process.Pid)
				if err := proc.process.Kill(); err != nil {
					log.Printf("Cannot kill the task process of Job(%s): %v\n", jobid, err)
				} else {
					node.executings[id] = TaskProcess{proc.process, true}
					log.Println("  Process killed")
				}
			}
		}
	}
}
