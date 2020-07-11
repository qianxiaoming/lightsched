package util

import "sync"

//WaitForStop 帮助使用WaitGroup等待一组任务完成
var WaitForStop = func(wg *sync.WaitGroup, wait func()) {
	wg.Add(1)
	defer wg.Done()
	wait()
}
