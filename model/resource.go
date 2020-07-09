package model

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/qianxiaoming/lightsched/util"
)

// ResourceCPU 是CPU资源的需求或提供信息
type ResourceCPU struct {
	Cores     float32 `json:"cores"`     // 使用的CPU个数，可以少于或多于1个CPU
	Frequency int     `json:"frequency"` // 使用的CPU主频，单位MHz
	MinFreq   int     `json:"min_freq"`  // 要求的最低CPU主频，单位MHz
}

// ResourceGPU 是GPU资源的需求或提供信息G
type ResourceGPU struct {
	Cards  int `json:"cards"`  // 使用的GPU个数
	Cores  int `json:"cores"`  // 要求的GPU最少核心数
	Memory int `json:"memory"` // 要求的GPU最低显存
	CUDA   int `json:"cuda"`   // 要求的CUDA最低版本，10.2为1020
}

// ResourceSet 是CPU、GPU及其它类型资源的总和
type ResourceSet struct {
	CPU    ResourceCPU    `json:"cpu"`
	GPU    ResourceGPU    `json:"gpu"`
	Memory int            `json:"memory"`
	Others map[string]int `json:"others,omitempty"`
}

// DefaultResourceSet 是赋给未指定任何资源需求任务的默认需求
var DefaultResourceSet *ResourceSet = &ResourceSet{
	CPU:    ResourceCPU{Cores: 0.8, Frequency: 2048},
	GPU:    ResourceGPU{Cards: 0},
	Memory: 1024,
}

// Clone 深度复制ResourceSet对象
func (res *ResourceSet) Clone() *ResourceSet {
	result := &ResourceSet{
		CPU:    res.CPU,
		GPU:    res.GPU,
		Memory: res.Memory,
	}
	if res.Others != nil {
		result.Others = make(map[string]int)
		for k, v := range res.Others {
			result.Others[k] = v
		}
	}
	return result
}

// SatisfiedWith 判断指定的资源集能否满足自身需求
func (res *ResourceSet) SatisfiedWith(other *ResourceSet) (bool, string, interface{}, interface{}) {
	if res.GPU.Cards > 0 {
		if res.GPU.Cards > other.GPU.Cards {
			return false, "GPU", res.GPU.Cards, other.GPU.Cards
		}
		if res.GPU.CUDA > other.GPU.CUDA {
			return false, "CUDA Version", float32(res.GPU.CUDA) / float32(100.0), float32(other.GPU.CUDA) / float32(100.0)
		}
		if res.GPU.Memory > other.GPU.Memory {
			return false, "GPU Memory", fmt.Sprintf("%dMi", res.GPU.Memory), fmt.Sprintf("%dMi", other.GPU.Memory)
		}
	}
	if res.CPU.MinFreq > 0 && res.CPU.MinFreq > other.CPU.MinFreq {
		return false, "CPU Minimum Frequency", fmt.Sprintf("%dMHz", res.CPU.MinFreq), fmt.Sprintf("%dMHz", other.CPU.MinFreq)
	}
	if res.CPU.Cores > 0 && res.CPU.Cores > other.CPU.Cores {
		return false, "CPU Cores", res.CPU.Cores, other.CPU.Cores
	}
	if res.CPU.Frequency > 0 && res.CPU.Frequency > other.CPU.Frequency {
		return false, "CPU Frequency", fmt.Sprintf("%dMHz", res.CPU.Frequency), fmt.Sprintf("%dMHz", other.CPU.Frequency)
	}
	if res.Memory > 0 && res.Memory > other.Memory {
		return false, "Memory", fmt.Sprintf("%fGi", float32(res.Memory)/float32(1000.0)), fmt.Sprintf("%fGi", float32(other.Memory)/float32(1000.0))
	}
	for k, v := range res.Others {
		if ov, ok := other.Others[k]; !ok {
			return false, k, v, 0
		} else if v > ov {
			return false, k, v, ov
		}
	}
	return true, "", nil, nil
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
		res.Memory = int(v)
	}
	res.Others = spec.Others
	return res
}
