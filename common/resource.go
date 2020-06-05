package common

// ResourceCPU 是CPU资源的需求或提供信息
type ResourceCPU struct {
	Cores     float32
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
