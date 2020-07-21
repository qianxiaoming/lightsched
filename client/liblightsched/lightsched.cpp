#include <boost/format.hpp>
#include "json/CJsonObject.hpp"
#include "lightsched.h"
#include "httputil.h"

namespace lightsched {

JobState ToJobState(const char* state)
{
	if (strcmp(state, "Queued") == 0)
		return JobState::Queued;
	if (strcmp(state, "Executing") == 0)
		return JobState::Executing;
	if (strcmp(state, "Halted") == 0)
		return JobState::Halted;
	if (strcmp(state, "Completed") == 0)
		return JobState::Completed;
	if (strcmp(state, "Failed") == 0)
		return JobState::Failed;
	return JobState::Terminated;
}

TaskState ToTaskState(const char* state)
{
	if (strcmp(state, "Queued") == 0)
		return TaskState::Queued;
	if (strcmp(state, "Scheduled") == 0)
		return TaskState::Scheduled;
	if (strcmp(state, "Dispatching") == 0)
		return TaskState::Dispatching;
	if (strcmp(state, "Executing") == 0)
		return TaskState::Executing;
	if (strcmp(state, "Completed") == 0)
		return TaskState::Completed;
	if (strcmp(state, "Failed") == 0)
		return TaskState::Failed;
	if (strcmp(state, "Aborted") == 0)
		return TaskState::Aborted;
	if (strcmp(state, "Terminated") == 0)
		return TaskState::Terminated;
	return TaskState::Queued;
}

NodeState ToNodeState(const char* state)
{
	if (strcmp(state, "Online") == 0)
		return NodeState::Online;
	if (strcmp(state, "Offline") == 0)
		return NodeState::Offline;
	return NodeState::Unknown;
}

ComputingCluster::ComputingCluster(std::string server, uint16_t port) : server_addr(server), httpclient(nullptr)
{
	server_port = boost::str(boost::format("%d") % port);
	httpclient = new HttpClient();
	std::string result;
	int code = HttpUtil::Get(server_addr + "/api/v1/healthz", server_port, result);
}

ComputingCluster::~ComputingCluster()
{
	delete httpclient;
	httpclient = nullptr;
}

bool ComputingCluster::IsConnected() const
{
	httpclient != nullptr;
}

std::string ComputingCluster::GetName() const
{

}

std::string ComputingCluster::GetServerAddr() const
{

}

bool ComputingCluster::SubmitJob(JobSpec& job_spec, std::string* errmsg)
{

}

bool ComputingCluster::TerminateJob(std::string id)
{

}

bool ComputingCluster::DeleteJob(std::string id)
{

}

JobPtr ComputingCluster::QueryJob(std::string id) const
{

}

JobList ComputingCluster::QueryJobList(JobState* state, int offset, int limits) const
{

}

NodeList ComputingCluster::GetNodeList() const
{

}

bool ComputingCluster::OfflineNode(std::string name)
{

}

bool ComputingCluster::OnlineNode(std::string name)
{

}

ResourceSet::ResourceSet() 
	: num_cpus(0.0), cpu_freq(0), memory(0), num_gpus(0), gpu_memory(0), cuda(0)
{
}

void ResourceSet::SetCPU(float count, int frequency)
{
	num_cpus = count;
	cpu_freq = frequency;
}

void ResourceSet::SetMemory(int megabytes)
{
	memory = megabytes;
}

void ResourceSet::SetGPU(int gpus, int min_memory, int min_cuda)
{
	num_gpus = gpus;
	gpu_memory = min_memory;
	cuda = min_cuda;
}

bool ResourceSet::IsNull() const
{
	return num_cpus == 0.0f && cpu_freq == 0 && num_gpus == 0 && memory == 0;
}

TaskSpec::TaskSpec()
{

}

TaskSpec::TaskSpec(std::string name) : task_name(name)
{
}

TaskSpec::TaskSpec(std::string name, std::string cmd, std::string cmd_args)
	: task_name(name), command(cmd), command_args(cmd_args)
{
}

JobSpec::JobSpec() : priority(1000), max_errors(0)
{
}

JobSpec::JobSpec(std::string name, const ResourceSet& res) 
	: job_name(name), priority(1000), resources(res)
{
}

JobSpec::JobSpec(std::string name, std::string cmd)
	: job_name(name), priority(1000), command(cmd)
{
}

JobSpec& JobSpec::AddTask(const TaskSpec& task)
{
	tasks.push_back(task);
	return *this;
}

JobSpec& JobSpec::AddTask(std::string name, std::string cmd, std::string cmd_args)
{
	TaskSpec spec(name, cmd, cmd_args);
	tasks.push_back(spec);
	return *this;
}

TaskInfo::TaskInfo() : task_state(TaskState::Queued), progress(0), start_time(0), finish_time(0), exit_code(0)
{
}

bool TaskInfo::IsFinished() const
{
	return task_state == TaskState::Aborted || task_state == TaskState::Completed ||
		task_state == TaskState::Failed || task_state == TaskState::Terminated;
}

NodeInfo::NodeInfo()
{

}

JobInfo::JobInfo()
{

}

Job::Job(ComputingCluster* c, std::string id)
{

}

bool Job::UpdateJobInfo(JobInfo& info)
{

}

TaskInfoList Job::GetTaskList()
{

}

TaskInfo Job::GetTask(std::string id)
{

}

bool Job::UpdateTaskInfo(TaskInfoList& tasks)
{

}

bool Job::TerminateTask(std::string task_id)
{

}

std::string Job::GetTaskLog(std::string task_id)
{

}

}
