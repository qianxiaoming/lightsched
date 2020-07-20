#ifndef LIGHTSCHED_CLIENT_API_H
#define LIGHTSCHED_CLIENT_API_H

#include <cstdint>
#include <map>
#include <string>
#include <memory>
#include <list>

#if defined(WIN32) || defined(_WINDOWS)
#if defined(LIBLIGHTSCHED_EXPORTS)
#define LIGHTSCHED_API __declspec(dllexport)
#else
#define LIGHTSCHED_API __declspec(dllimport)
#pragma comment(lib,"liblightsched.lib")
#endif
#pragma warning(disable: 4251 4275)
#else
#define LIGHTSCHED_API
#endif

namespace lightsched {

typedef std::map<std::string, std::string> LabelList;

enum class TaskState { Queued, Scheduled, Dispatching, Executing, Completed, Failed, Aborted, Terminated };
enum class JobState { Queued, Executing, Halted, Completed, Failed, Terminated };
enum class NodeState { Online, Offline, Unknown };

const uint16_t APISERVER_PORT = 20516;

class Job;
struct JobSpec;
struct TaskSpec;
struct TaskInfo;
struct NodeInfo;
typedef std::shared_ptr<Job> JobPtr;
typedef std::list<JobPtr> JobList;
typedef std::list<TaskSpec> TaskSpecList;
typedef std::list<NodeInfo> NodeList;

class LIGHTSCHED_API ComputingCluster
{
public:
	ComputingCluster(std::string server, uint16_t port);

	bool IsConnected() const;

	std::string GetName() const;

	std::string GetServerAddr() const;

	bool SubmitJob(JobSpec& job_spec, std::string* errmsg = nullptr);

	bool TerminateJob(std::string id);

	bool DeleteJob(std::string id);

	JobPtr QueryJob(std::string id) const;

	JobList QueryJobList(JobState* state = nullptr, int offset = 0, int limits = -1) const;

	NodeList GetNodeList() const;

	bool OfflineNode(std::string name);

	bool OnlineNode(std::string name);

private:
	std::string server_addr;
	std::string cluster_name;
	bool        connected;
};

struct LIGHTSCHED_API ResourceClaim
{

};

struct LIGHTSCHED_API TaskSpec
{
	TaskSpec();
	TaskSpec(std::string name);
	TaskSpec(std::string name, std::string cmd, std::string cmd_args);

	std::string   task_name;
	std::string   command;
	std::string   command_args;
	std::string   environments;
	std::string   labels;
	std::string   work_dir;
	ResourceClaim resources;
};

struct LIGHTSCHED_API JobSpec
{
	JobSpec();
	JobSpec(std::string name, const ResourceClaim& claims);
	JobSpec(std::string name, std::string cmd);
	JobSpec& AddTask(const TaskSpec& task);
	JobSpec& AddTask(std::string name, std::string cmd, std::string cmd_args);

	std::string   job_id;
	std::string   job_name;
	std::string   environments;
	std::string   labels;
	int           priority;
	int           max_errors;
	std::string   command;
	std::string   work_dir;
	ResourceClaim resources;
	TaskSpecList  tasks;
};

struct LIGHTSCHED_API TaskInfo
{
	TaskInfo();
	bool IsFinished() const;

	std::string task_id;
	std::string task_name;
	TaskState   task_state;
	int32_t     progress;
	std::string message;
	std::string exec_node;
	time_t      start_time;
	time_t      finish_time;
	uint32_t    exit_code;
};
typedef std::list<TaskInfo> TaskInfoList;

struct PlatformInfo
{
	std::string kind;
	std::string os;
	std::string family;
	std::string version;
};

struct LIGHTSCHED_API NodeInfo
{
	NodeInfo();

	std::string   name;
	std::string   address;
	PlatformInfo  platform;
	NodeState     state;
	time_t        online;
	std::string   labels;
	ResourceClaim resources;
};

struct LIGHTSCHED_API JobInfo
{
	JobInfo();

	JobState          job_state;
	int32_t           progress;
	int32_t           total_tasks;
	time_t            submit_time;
	time_t            exec_time;
	time_t            finish_time;
};

class LIGHTSCHED_API Job
{
public:
	Job(ComputingCluster* c, std::string id);

	const JobSpec& GetSpec() const { return job_spec; }

	JobSpec& GetSpec() { return job_spec; }

	bool UpdateJobInfo(JobInfo& info);

	const JobInfo& GetJobInfo() const { return job_info; }

	bool Halt();

	bool Resume();

	// get all tasks
	TaskInfoList GetTaskList();

	TaskInfo GetTask(std::string id);

	// update task status in passed list
	bool UpdateTaskInfo(TaskInfoList& tasks);

	bool TerminateTask(std::string task_id);

	std::string GetTaskLog(std::string task_id);

private:
	ComputingCluster* cluster;
	JobSpec           job_spec;
	JobInfo           job_info;
};

}

#endif
