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
	hb := &message.Heartbeat{
		Name:   node.config.hostname,
		CPU:    cpu[0],
		Memory: mem.UsedPercent,
	}
	request, _ := json.Marshal(hb)
	if resp, err := http.Post(node.heartbeat.url, "application/json", bytes.NewReader(request)); err != nil {
		log.Printf("Send heartbeat failed: %v\n", err)
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
			for i, msg := range msgs {
				log.Println("    ", i, msg.Kind)
			}
		}
	}
	return nil
}
