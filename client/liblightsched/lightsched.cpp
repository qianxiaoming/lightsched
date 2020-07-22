#include <boost/format.hpp>
#include <boost/algorithm/string.hpp>
#include <boost/date_time.hpp>
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

const char* ToString(JobState state)
{
	switch (state) {
	case JobState::Queued:
		return "Queued";
	case JobState::Executing:
		return "Executing";
	case JobState::Halted:
		return "Halted";
	case JobState::Completed:
		return "Completed";
	case JobState::Failed:
		return "Failed";
	case JobState::Terminated:
		return "Terminated";
	}
	return "";
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
	try {
		httpclient = new HttpClient();
		if (httpclient->Connect(server_addr, port)) {
			std::string result;
			if (httpclient->Get("/cluster", result) == http::status::ok) {
				neb::CJsonObject json(result);
				cluster_name = json("id");
				return;
			}
		}
	} catch (std::exception const& e) {
		std::cerr << "Error in connecting to " << server_addr << ": " << e.what() << std::endl;
	}
	delete httpclient;
	httpclient = nullptr;
}

ComputingCluster::~ComputingCluster()
{
	delete httpclient;
	httpclient = nullptr;
}

bool ComputingCluster::IsConnected() const
{
	return httpclient != nullptr;
}

std::string ComputingCluster::GetName() const
{
	return cluster_name;
}

std::string ComputingCluster::GetServerAddr() const
{
	return server_addr;
}

HttpClient* ComputingCluster::GetHttpClient() const
{
	return httpclient;
}

static void SetResourceSet(neb::CJsonObject& obj, const ResourceSet& res)
{
	obj.AddEmptySubObject("resources");
	obj["resources"].AddEmptySubObject("cpu");
	if (res.num_cpus > 0)
		obj["resources"]["cpu"].Add("cores", boost::str(boost::format("%.1f") % res.num_cpus));
	if (res.cpu_freq > 0)
		obj["resources"]["cpu"].Add("frequency", boost::str(boost::format("%.1fGHz") % (res.cpu_freq / 1000.0f)));
	if (res.memory > 0)
		obj["resources"].Add("memory", boost::str(boost::format("%dMi") % res.memory));
	obj["resources"].AddEmptySubObject("gpu");
	if (res.num_gpus > 0)
		obj["resources"]["gpu"].Add("cards", boost::str(boost::format("%d") % res.num_gpus));
	if (res.gpu_memory > 0)
		obj["resources"]["gpu"].Add("memory", boost::str(boost::format("%dGi") % res.gpu_memory));
	if (res.cuda > 0)
		obj["resources"]["gpu"].Add("cuda", boost::str(boost::format("%d") % res.cuda));
	if (!res.others.empty()) {
		obj["resources"].AddEmptySubObject("others");
		for (std::map<std::string, std::string>::const_iterator it = res.others.begin();
			it != res.others.end(); it++) {
			obj["resources"]["gpu"].Add(it->first, it->second);
		}
	}
}

static void GetResourceSet(ResourceSet& res, neb::CJsonObject& obj)
{
	res.num_cpus = std::atof(obj["cpu"]("cores").c_str());
	res.cpu_freq = std::atoi(obj["cpu"]("frequency").c_str());
	res.memory = std::atoi(obj("memory").c_str());
	res.num_gpus = std::atoi(obj["gpu"]("cards").c_str());
	res.gpu_memory = std::atoi(obj["gpu"]("memory").c_str());
	res.cuda = std::atoi(obj["gpu"]("cuda").c_str());
}

bool ComputingCluster::SubmitJob(JobSpec& job_spec, std::string* errmsg)
{
	if (job_spec.tasks.empty()) {
		std::cerr << "No task found" << std::endl;
		return false;
	}

	neb::CJsonObject spec;
	if (!job_spec.job_id.empty())
		spec.Add("id", job_spec.job_id);
	spec.Add("name", job_spec.job_name);
	spec.Add("queue", "default");
	spec.Add("priority", job_spec.priority);
	spec.Add("max_errors", job_spec.max_errors);
	if (!job_spec.labels.empty()) {
		spec.AddEmptySubObject("labels");
		for (LabelList::iterator it = job_spec.labels.begin(); it != job_spec.labels.end(); it++) {
			spec["labels"].Add(it->first, it->second);
		}
	}
	spec.AddEmptySubArray("groups");

	neb::CJsonObject group;
	group.Add("name", "main");
	if (!job_spec.environments.empty()) {
		std::vector<std::string> envs;
		boost::algorithm::split(envs, job_spec.environments, [](char c) { return c == ';'; });
		spec["groups"].AddEmptySubArray("envs");
		for (size_t i = 0; i < envs.size(); i++)
			spec["groups"]["envs"].Add(envs[i]);
	}
	if (!job_spec.command.empty())
		group.Add("command", job_spec.command);
	if (!job_spec.work_dir.empty())
		group.Add("workdir", job_spec.work_dir);
	if (!job_spec.resources.IsNull())
		SetResourceSet(group, job_spec.resources);
	spec["groups"].Add(group);

	spec["groups"][0].AddEmptySubArray("tasks");
	for (TaskSpecList::iterator it = job_spec.tasks.begin(); it != job_spec.tasks.end(); it++) {
		neb::CJsonObject task;
		task.Add("name", it->task_name);
		if (!it->command.empty())
			task.Add("command", it->command);
		if (!it->command_args.empty())
			task.Add("args", it->command_args);
		if (!it->work_dir.empty())
			task.Add("workdir", it->work_dir);
		if (!it->environments.empty()) {
			std::vector<std::string> envs;
			boost::algorithm::split(envs, it->environments, [](char c) { return c == ';'; });
			task.AddEmptySubArray("envs");
			for (size_t i = 0; i < envs.size(); i++)
				task["envs"].Add(envs[i]);
		}
		if (!it->labels.empty()) {
			task.AddEmptySubObject("labels");
			for (LabelList::iterator lab = it->labels.begin(); lab != it->labels.end(); lab++) {
				task["labels"].Add(lab->first, lab->second);
			}
		}
		if (!it->resources.IsNull())
			SetResourceSet(task, it->resources);
		spec["groups"][0]["tasks"].Add(task);
	}
	try {
		std::string result;
		if (httpclient->Post("/jobs", spec.ToString(), result) == http::status::created) {
			neb::CJsonObject json(result);
			job_spec.job_id = json("id");
		}
		else if (errmsg != nullptr) {
			*errmsg = result;
			std::cerr << spec.ToFormattedString() << std::endl;
			return false;
		}
	}
	catch (std::exception const& e) {
		if (errmsg != nullptr)
			*errmsg = e.what();
		return false;
	}
	return true;
}

bool ComputingCluster::TerminateJob(std::string id)
{
	try {
		std::string result;
		if (httpclient->Put(boost::str(boost::format("/jobs/%s/_terminate") % id), std::string(""), result) != http::status::ok) {
			std::cerr << result << std::endl;
			return false;
		}
	}
	catch (std::exception const& e) {
		std::cerr << e.what() << std::endl;
		return false;
	}
	return true;
}

bool ComputingCluster::DeleteJob(std::string id)
{
	try {
		std::string result;
		if (httpclient->Delete(boost::str(boost::format("/jobs/%s") % id), result) != http::status::ok) {
			std::cerr << result << std::endl;
			return false;
		}
	}
	catch (std::exception const& e) {
		std::cerr << e.what() << std::endl;
		return false;
	}
	return true;
}

JobPtr ComputingCluster::QueryJob(std::string id)
{
	JobPtr job;
	try {
		std::string result;
		if (httpclient->Get(boost::str(boost::format("/jobs/%s") % id), result) != http::status::ok)
			return job;
		neb::CJsonObject json(result);
		job.reset(new Job(this, id));
		JobSpec& spec = job->GetSpec();
		spec.job_name = json("name");
		spec.max_errors = std::atoi(json("max_errors").c_str());
		spec.priority = std::atoi(json("priority").c_str());
		JobInfo& info = job->GetJobInfo();
		info.total_tasks = std::atoi(json("tasks").c_str());
		info.job_state = JobState(std::atoi(json("state").c_str()));
		info.progress = std::atoi(json("progress").c_str());
		info.submit_time = json("submit_time");
		info.exec_time = json("exec_time");
		info.finish_time = json("finish_time");
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get job: " << e.what() << std::endl;
	}
	return job;
}

JobList ComputingCluster::QueryJobList(JobState* state, int offset, int limits)
{
	JobList jobs;
	try {
		std::string url = "/jobs?", strstate, stroffset, strlimits;
		if (state != nullptr)
			url += boost::str(boost::format("state=%s&") % ToString(*state));
		if (offset > 0)
			url += boost::str(boost::format("offset=%d&") % offset);
		if (limits > 0)
			url += boost::str(boost::format("limits=%d") % limits);
		std::string result;
		if (httpclient->Get(url, result) != http::status::ok)
			return jobs;
		neb::CJsonObject json(result);
		int count = json.GetArraySize();
		for (int i = 0; i < count; i++) {
			JobPtr job(new Job(this, json[i]("id")));
			JobSpec& spec = job->GetSpec();
			spec.job_name = json[i]("name");
			spec.max_errors = std::atoi(json[i]("max_errors").c_str());
			spec.priority = std::atoi(json[i]("priority").c_str());
			JobInfo& info = job->GetJobInfo();
			info.total_tasks = std::atoi(json[i]("tasks").c_str());
			info.job_state = JobState(std::atoi(json[i]("state").c_str()));
			info.progress = std::atoi(json[i]("progress").c_str());
			info.submit_time = json[i]("submit_time");
			info.exec_time = json[i]("exec_time");
			info.finish_time = json[i]("finish_time");
			jobs.push_back(job);
		}
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get job list: " << e.what() << std::endl;
	}
	return jobs;
}

NodeList ComputingCluster::GetNodeList()
{
	NodeList nodes;
	try {
		std::string result;
		if (httpclient->Get("/nodes", result) != http::status::ok)
			return nodes;
		neb::CJsonObject json(result);
		int count = json.GetArraySize();
		for (int i = 0; i < count; i++) {
			NodeInfo node;
			node.name = json[i]("name");
			node.address = json[i]("address");
			node.platform.kind = json[i]["platform"]("kind");
			node.platform.name = json[i]["platform"]("name");
			node.platform.family = json[i]["platform"]("family");
			node.platform.version = json[i]["platform"]("version");
			node.state = NodeState(std::atoi(json[i]("state").c_str()));
			node.online = json[i]("online");
			GetResourceSet(node.resources, json[i]["resources"]);
			nodes.push_back(node);
		}
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get node list: " << e.what() << std::endl;
	}
	return nodes;
}

bool ComputingCluster::OfflineNode(std::string name)
{
	try {
		std::string result;
		if (httpclient->Put(boost::str(boost::format("/nodes/%s/_offline") % name), std::string(""), result) != http::status::ok) {
			std::cerr << result << std::endl;
			return false;
		}
	}
	catch (std::exception const& e) {
		std::cerr << e.what() << std::endl;
		return false;
	}
	return true;
}

bool ComputingCluster::OnlineNode(std::string name)
{
	try {
		std::string result;
		if (httpclient->Put(boost::str(boost::format("/nodes/%s/_online") % name), std::string(""), result) != http::status::ok) {
			std::cerr << result << std::endl;
			return false;
		}
	}
	catch (std::exception const& e) {
		std::cerr << e.what() << std::endl;
		return false;
	}
	return true;
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

void ResourceSet::SetGPU(int gpus, int gigabytes, int min_cuda)
{
	num_gpus = gpus;
	gpu_memory = gigabytes;
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
	: job_name(name), priority(1000), max_errors(0), resources(res)
{
}

JobSpec::JobSpec(std::string name, std::string cmd)
	: job_name(name), priority(1000), max_errors(0), command(cmd)
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

TaskInfo::TaskInfo() 
	: task_state(TaskState::Queued), progress(0), exit_code(0)
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
	cluster = c;
	job_spec.job_id = id;
}

bool Job::UpdateJobInfo(JobInfo& info)
{
	JobPtr job;
	try {
		std::string result;
		if (cluster->GetHttpClient()->Get(boost::str(boost::format("/jobs/%s") % job_spec.job_id), result) != http::status::ok)
			return false;
		neb::CJsonObject json(result);
		info.total_tasks = std::atoi(json("tasks").c_str());
		info.job_state = JobState(std::atoi(json("state").c_str()));
		info.progress = std::atoi(json("progress").c_str());
		info.submit_time = json("submit_time");
		info.exec_time = json("exec_time");
		info.finish_time = json("finish_time");
	}
	catch (std::exception const& e) {
		std::cerr << "Error in update job info: " << e.what() << std::endl;
	}
	return true;
}

TaskInfoList Job::GetTaskList()
{
	TaskInfoList tasks;
	try {
		std::string result;
		if (cluster->GetHttpClient()->Get(boost::str(boost::format("/tasks?jobid=%s") % job_spec.job_id), result) != http::status::ok)
			return tasks;
		neb::CJsonObject json(result);
		int count = json.GetArraySize();
		for (int i = 0; i < count; i++) {
			TaskInfo task;
			task.task_id = json[i]("id");
			task.task_name = json[i]("name");
			task.task_state = TaskState(std::atoi(json[i]("state").c_str()));
			task.exec_node = json[i]("node");
			task.progress = std::atoi(json[i]("progress").c_str());
			task.start_time = json[i]("start_time");
			task.finish_time = json[i]("finish_time");
			task.exit_code = std::atoi(json[i]("exit_code").c_str());
			task.message = json[i]("error");
			tasks.push_back(task);
		}
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get task list: " << e.what() << std::endl;
	}
	return tasks;
}

bool Job::GetTask(std::string task_id, TaskInfo& info)
{
	try {
		std::string result;
		if (cluster->GetHttpClient()->Get(boost::str(boost::format("/tasks/%s") % task_id), result) != http::status::ok)
			return false;
		neb::CJsonObject json(result);
		info.task_id = json("id");
		info.task_name = json("name");
		info.task_state = TaskState(std::atoi(json("state").c_str()));
		info.exec_node = json("node");
		info.progress = std::atoi(json("progress").c_str());
		info.start_time = json("start_time");
		info.finish_time = json("finish_time");
		info.exit_code = std::atoi(json("exit_code").c_str());
		info.message = json("error");
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get task list: " << e.what() << std::endl;
		return false;
	}
	return true;
}

bool Job::UpdateTaskInfo(TaskInfoList& tasks)
{
	if (tasks.empty())
		return false;
	std::ostringstream oss;
	for (TaskInfoList::iterator it = tasks.begin(); it != tasks.end(); it++) {
		oss << it->task_id << ",";
	}
	std::string ids = oss.str();
	if (ids[ids.length() - 1] == ',')
		ids.erase(ids.length() - 1, 1);

	try {
		std::string result;
		if (cluster->GetHttpClient()->Get(boost::str(boost::format("/tasks?ids=%s") % ids), result) != http::status::ok)
			return false;
		neb::CJsonObject json(result);
		int count = json.GetArraySize();
		for (int i = 0; i < count; i++) {
			std::string curid = json[i]("id");
			TaskInfoList::iterator it = tasks.begin();
			while (it != tasks.end()) {
				if (it->task_id == curid)
					break;
				it++;
			}
			if (it == tasks.end())
				continue;

			TaskInfo& task = *it;
			task.task_name = json[i]("name");
			task.task_state = TaskState(std::atoi(json[i]("state").c_str()));
			task.exec_node = json[i]("node");
			task.progress = std::atoi(json[i]("progress").c_str());
			task.start_time = json[i]("start_time");
			task.finish_time = json[i]("finish_time");
			task.exit_code = std::atoi(json[i]("exit_code").c_str());
			task.message = json[i]("error");
		}
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get task list: " << e.what() << std::endl;
	}
	return true;
}

std::string Job::GetTaskLog(std::string task_id)
{
	try {
		std::string result;
		if (cluster->GetHttpClient()->Get(boost::str(boost::format("/tasks/%s/log") % task_id), result) != http::status::ok)
			return "";
		return result;
	}
	catch (std::exception const& e) {
		std::cerr << "Error in get task log: " << e.what() << std::endl;
		return "";
	}
	return "";
}

}
