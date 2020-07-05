package model

import (
	"log"
	"strconv"
	"strings"

	"github.com/qianxiaoming/lightsched/util"
)

// ResourceCPU 是CPU资源的需求或提供信息
type ResourceCPU struct {
	Cores     float32 // 使用的CPU个数，可以少于或多于1个CPU
	Frequency int     // 使用的CPU主频，单位MHz
	MinFreq   int     // 要求的最低CPU主频，单位MHz
}

// ResourceGPU 是GPU资源的需求或提供信息G
type ResourceGPU struct {
	Cards  int // 使用的GPU个数
	Cores  int // 要求的GPU最少核心数
	Memory int // 要求的GPU最低显存
	CUDA   int // 要求的CUDA最低版本，10.2为1020
}

// ResourceMem 是内存资源的需求或提供信息
type ResourceMem struct {
	Total int
	Used  int
	Free  int
}

// ResourceSet 是CPU、GPU及其它类型资源的总和
type ResourceSet struct {
	CPU    ResourceCPU
	GPU    ResourceGPU
	Memory ResourceMem
	Others map[string]int
}

// DefaultResourceSet 是赋给未指定任何资源需求任务的默认需求
var DefaultResourceSet *ResourceSet = &ResourceSet{
	CPU:    ResourceCPU{Cores: 0.8, Frequency: 2048},
	GPU:    ResourceGPU{Cards: 0},
	Memory: ResourceMem{Total: 1024},
}

// ResourceSpec 是提交作业时指定的资源信息，资源值可以包含单位
// 使用string作为指定的值允许用户指定使用资源量的单位
type ResourceSpec struct {
	CPU    map[string]string `json:"cpu"`
	GPU    map[string]string `json:"gpu"`
	Memory string            `json:"memory"`
	Others map[string]int    `json:"others"`
}

// NewResourceSetWithSpec 根据指定的资源信息创建ResourceSet对象
func NewResourceSetWithSpec(spec *ResourceSpec) *ResourceSet {
	if spec == nil {
		return nil
	}
	res := &ResourceSet{}
	// 解析CPU资源需求
	for k, v := range spec.CPU {
		k = strings.ToLower(k)
		v = strings.Trim(v, " ")
		if strings.Compare(k, "cores") == 0 {
			cores, err := strconv.ParseFloat(v, 32)
			if err != nil {
				res.CPU.Cores = DefaultResourceSet.CPU.Cores
				log.Printf("Unable to pares the number of cores of CPU resource: %v", err)
			} else {
				res.CPU.Cores = float32(cores)
			}
		} else if strings.Compare(k, "frequency") == 0 {
			v, u := util.ParseValueAndUnit(v)
			if strings.Compare(u, "ghz") == 0 {
				v = v * 1000
			}
			res.CPU.Frequency = int(v)
		} else if strings.Compare(k, "min_frequency") == 0 {
			v, u := util.ParseValueAndUnit(v)
			if strings.Compare(u, "ghz") == 0 {
				v = v * 1000
			}
			res.CPU.MinFreq = int(v)
		}
	}
	// 解析GPU资源需求
	for k, v := range spec.GPU {
		k = strings.ToLower(k)
		v = strings.Trim(v, " ")
		if strings.Compare(k, "cards") == 0 {
			cards, _ := strconv.Atoi(v)
			res.GPU.Cards = cards
		} else if strings.Compare(k, "cores") == 0 {
			cores, _ := strconv.Atoi(v)
			res.GPU.Cores = cores
		} else if strings.Compare(k, "memory") == 0 {
			v, u := util.ParseValueAndUnit(v)
			if strings.Compare(u, "gi") == 0 {
				v = v * 1000
			}
			res.GPU.Memory = int(v)
		} else if strings.Compare(k, "cuda") == 0 {
			v, err := strconv.ParseFloat(v, 32)
			if err == nil {
				res.GPU.CUDA = int(v * 100)
			}
		}
	}
	// 解析内存资源
	if len(spec.Memory) > 0 {
		v, u := util.ParseValueAndUnit(spec.Memory)
		if strings.Compare(u, "gi") == 0 {
			v = v * 1000
		}
		res.Memory.Total = int(v)
	}
	res.Others = spec.Others
	return res
}
