#include "lightsched.h"

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

ComputingCluster::ComputingCluster(std::string server, uint16_t port)
{

}

bool ComputingCluster::IsConnected() const
{

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

TaskSpec::TaskSpec()
{

}

TaskSpec::TaskSpec(std::string name)
{

}

TaskSpec::TaskSpec(std::string name, std::string cmd, std::string cmd_args)
{

}

JobSpec::JobSpec()
{

}

JobSpec::JobSpec(std::string name, const ResourceSet& res)
{

}

JobSpec::JobSpec(std::string name, std::string cmd)
{

}

JobSpec& JobSpec::AddTask(const TaskSpec& task)
{

}

JobSpec& JobSpec::AddTask(std::string name, std::string cmd, std::string cmd_args)
{

}

TaskInfo::TaskInfo()
{

}

bool TaskInfo::IsFinished() const
{

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
