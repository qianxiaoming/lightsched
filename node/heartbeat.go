package node

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

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
	for _, msg := range msgs {
		switch msg.Kind {
		case message.KindScheduleTask:
			go node.runExecuteTask(msg)
		}
	}
}
