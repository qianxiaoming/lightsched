package util

import (
	"strings"
	"sync"

	uuid "github.com/satori/go.uuid"
)

//WaitForStop 帮助使用WaitGroup等待一组任务完成
var WaitForStop = func(wg *sync.WaitGroup, wait func()) {
	wg.Add(1)
	defer wg.Done()
	wait()
}

// GenerateUUID 生成1个UUID字符串
func GenerateUUID() string {
	id := uuid.NewV4().String()
	id = strings.ReplaceAll(id, "-", "")
	// 将32个字节缩短为16个字节
	var sb strings.Builder
	for i, c := range id {
		if i&0x01 == 0 {
			sb.WriteRune(c)
		}
	}
	return sb.String()
}
