package util

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PathExists 判断路径是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else {
		if !os.IsNotExist(err) {
			log.Printf("%v", err)
		}
		return false
	}
}

// MakeDirAll 在目录不存在时创建目录
func MakeDirAll(dirPath string) error {
	exists := PathExists(dirPath)
	if !exists {
		err := os.MkdirAll(dirPath, 0777)
		return err
	}
	return nil
}

// UniformPath 用于规范化文件路径
func UniformPath(p string) string {
	p = strings.Replace(p, "\\", "/", -1)
	p = path.Clean(p)
	p = strings.TrimSuffix(p, "/")
	return p
}

// GetCurrentPath 用于获取程序的当前路径
func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}
