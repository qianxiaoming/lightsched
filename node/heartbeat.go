package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/qianxiaoming/lightsched/message"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var errNodeNotRegistered error = errors.New("node not registered")

func (node *NodeServer) sendHeartbeat() error {
	cpu, _ := cpu.Percent(0, false)
	mem, _ := mem.VirtualMemory()
	var payload []*message.TaskReport
	if len(node.heartbeat.payload) > 0 {
		payload = make([]*message.TaskReport, 0, len(node.heartbeat.payload))
		for _, v := range node.heartbeat.payload {
			payload = append(payload, v)
		}
		node.heartbeat.payload = make(map[string]*message.TaskReport)
	}
	hb := &message.Heartbeat{
		Name:       node.config.Hostname,
		CPU:        cpu[0],
		Memory:     mem.UsedPercent,
		Executings: len(node.executings),
		Payload:    payload,
	}
	if request, err := json.Marshal(hb); err != nil {
		log.Printf("Failed to marshal heartbeat message with %d payload(s): %T %+v\n", len(payload), err, err)
		log.Printf("%v\n", hb)
		return err
	} else {
		if resp, err := http.Post(node.heartbeat.url, "application/json", bytes.NewReader(request)); err != nil {
			log.Printf("Send heartbeat failed with body length %d: %T %+v\n", len(request), err, err)
			log.Printf("%s\n", string(request))
			// 心跳发送失败时需要恢复原来的待发送信息
			for _, status := range payload {
				node.heartbeat.payload[status.ID] = status
			}
			return err
		} else {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			if len(body) != 0 {
				msgs := make([]*message.JSON, 0)
				if err = json.Unmarshal(body, &msgs); err != nil {
					log.Printf("Unable to unmarshal the response for heartbeat: %v\n", err)
				} else {
					node.runServerMessages(msgs)
				}
			}
			if resp.StatusCode == http.StatusNotFound {
				// 节点需要重新注册自己
				log.Println("Not found this node in API Server, register self now")
				return errNodeNotRegistered
			}
		}
	}
	return nil
}

func (node *NodeServer) runServerMessages(msgs []*message.JSON) {
	for _, msg := range msgs {
		switch msg.Kind {
		case message.KindScheduleTask:
			go node.runExecuteTask(msg)
		case message.KindTerminateJob:
			jobid := msg.Object
			log.Printf("Terminating job %s...\n", jobid)
			for id, proc := range node.executings {
				if strings.HasPrefix(id, jobid) && proc.process != nil {
					log.Printf("  Killing task(%s) process %d...\n", id, proc.process.Pid)
					if err := proc.process.Kill(); err != nil {
						log.Printf("Cannot kill the task process of Job(%s): %v\n", jobid, err)
					} else {
						node.executings[id] = TaskProcess{proc.process, true}
						log.Printf("  Process %d killed\n", proc.process.Pid)
					}
				}
			}
		}
	}
}
