package model

// ResourceCPU 是CPU资源的需求或提供信息
type ResourceCPU struct {
	Cores     int // CPU核数总是乘以10使用，将0.5个CPU转为整数
	Frequency int
	MinFreq   int
}

// ResourceGPU 是GPU资源的需求或提供信息
type ResourceGPU struct {
	Cards  int
	Cores  int
	Memory int
	CUDA   int
}

// ResourceSet 是CPU、GPU及其它类型资源的总和
type ResourceSet struct {
	CPU      ResourceCPU
	GPU      ResourceGPU
	Generics map[string]int
}

// ResourceSetSpec 是提交作业时指定的资源信息，资源值可以包含单位
type ResourceSetSpec struct {
	CPU      map[string]interface{}
	GPU      map[string]interface{}
	Generics map[string]interface{}
}
