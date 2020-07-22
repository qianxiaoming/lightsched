#ifndef LIGHTSCHED_CLIENT_API_H
#define LIGHTSCHED_CLIENT_API_H

#include <cstdint>
#include <map>
#include <vector>
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

class HttpClient;
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
	ComputingCluster(std::string server, uint16_t port = APISERVER_PORT);

	~ComputingCluster();

	bool IsConnected() const;

	HttpClient* GetHttpClient() const;

	std::string GetName() const;

	std::string GetServerAddr() const;

	bool SubmitJob(JobSpec& job_spec, std::string* errmsg = nullptr);

	bool TerminateJob(std::string id);

	bool DeleteJob(std::string id);

	JobPtr QueryJob(std::string id);

	JobList QueryJobList(JobState* state = nullptr, int offset = 0, int limits = -1);

	NodeList GetNodeList();

	bool OfflineNode(std::string name);

	bool OnlineNode(std::string name);

private:
	std::string server_addr;
	std::string cluster_name;
	HttpClient* httpclient;
};

struct LIGHTSCHED_API ResourceSet
{
	ResourceSet();
	void SetCPU(float count, int frequency = 0);
	void SetMemory(int megabytes);
	void SetGPU(int gpus, int gigabytes, int min_cuda);
	bool IsNull() const;

	float num_cpus;
	int cpu_freq;
	int memory;
	int num_gpus;
	int gpu_memory;
	int cuda;
	std::map<std::string, std::string> others;
};

struct LIGHTSCHED_API TaskSpec
{
	TaskSpec();
	TaskSpec(std::string name);
	TaskSpec(std::string name, std::string cmd, std::string cmd_args);

	std::string task_name;
	std::string command;
	std::string command_args;
	std::string environments;
	std::string work_dir;
	LabelList   labels;
	ResourceSet resources;
};

struct LIGHTSCHED_API JobSpec
{
	JobSpec();
	JobSpec(std::string name, const ResourceSet& res);
	JobSpec(std::string name, std::string cmd);
	JobSpec& AddTask(const TaskSpec& task);
	JobSpec& AddTask(std::string name, std::string cmd, std::string cmd_args);

	std::string  job_id;
	std::string  job_name;
	std::string  environments;
	LabelList    labels;
	int          priority;
	int          max_errors;
	std::string  command;
	std::string  work_dir;
	ResourceSet  resources;
	TaskSpecList tasks;
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
	std::string start_time;
	std::string finish_time;
	uint32_t    exit_code;
};
typedef std::list<TaskInfo> TaskInfoList;

struct PlatformInfo
{
	std::string kind;
	std::string name;
	std::string family;
	std::string version;
};

struct LIGHTSCHED_API NodeInfo
{
	NodeInfo();

	std::string  name;
	std::string  address;
	PlatformInfo platform;
	NodeState    state;
	std::string  online;
	LabelList    labels;
	ResourceSet  resources;
};

struct LIGHTSCHED_API JobInfo
{
	JobInfo();

	JobState    job_state;
	int32_t     progress;
	int32_t     total_tasks;
	std::string submit_time;
	std::string exec_time;
	std::string finish_time;
};

class LIGHTSCHED_API Job
{
public:
	Job(ComputingCluster* c, std::string id);

	const JobSpec& GetSpec() const { return job_spec; }

	JobSpec& GetSpec() { return job_spec; }

	bool UpdateJobInfo(JobInfo& info);

	const JobInfo& GetJobInfo() const { return job_info; }

	JobInfo& GetJobInfo() { return job_info; }

	TaskInfoList GetTaskList();

	bool GetTask(std::string task_id, TaskInfo& info);

	bool UpdateTaskInfo(TaskInfoList& tasks);

	std::string GetTaskLog(std::string task_id);

private:
	ComputingCluster* cluster;
	JobSpec           job_spec;
	JobInfo           job_info;
};

LIGHTSCHED_API const char* ToString(JobState state);
LIGHTSCHED_API JobState ToJobState(const char* state);
LIGHTSCHED_API TaskState ToTaskState(const char* state);

}

#endif
