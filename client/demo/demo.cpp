#include <iostream>
#include <lightsched.h>
#include <windows.h>

int main()
{
    using namespace lightsched;
    ComputingCluster* cluster = new ComputingCluster("127.0.0.1");
    if (!cluster->IsConnected()) {
        std::cout << "Cannot connect to server" << std::endl;
        delete cluster;
        return 1;
    } else
        std::cout << "Cluster " << cluster->GetName() << " connected" << std::endl<<std::endl;
    std::cout << "=================    Nodes    =================" << std::endl;
    NodeList nodes = cluster->GetNodeList();
    for (NodeList::iterator it = nodes.begin(); it != nodes.end(); it++) {
        NodeInfo node = *it;
        std::cout << "  " << node.name << "\t\t" << node.address << "\t\t" << node.platform.name << "\t"
            << node.online << "\t" << node.resources.num_cpus << " CPUs," << node.resources.num_gpus << " GPUs" << std::endl;
    }
    std::cout << std::endl;
    std::cout << "================= Submit Jobs =================" << std::endl;
    JobSpec jobspec("Hello", "c:/windows/my.exe");
    TaskSpec task;
    task.task_name = "TASK-1";
    task.command = "c:/Develop/model.exe";
    task.command_args = "-f test1.dat -n 10";
    task.resources.num_cpus = 1.8f;
    task.resources.memory = 4000;
    task.resources.num_gpus = 1;
    task.resources.gpu_memory = 4;
    jobspec.AddTask(task);
    task.task_name = "TASK-2";
    task.command_args = "-f test2.dat -n 20";
    task.resources.num_cpus = 2.0f;
    jobspec.AddTask(task);
    task.task_name = "TASK-3";
    task.command_args = "-f test3.dat -n 30";
    task.resources.num_cpus = 3.0f;
    task.resources.num_gpus = 0;
    task.resources.gpu_memory = 0;
    jobspec.AddTask(task);
    task.task_name = "TASK-4";
    task.command_args = "-f test4.dat -n 40";
    task.resources.num_cpus = 1.0f;
    task.resources.num_gpus = 1;
    task.resources.gpu_memory = 4;
    jobspec.AddTask(task);
    task.task_name = "TASK-5";
    task.command_args = "-f test5.dat -n 50";
    task.resources.num_cpus = 4.0f;
    jobspec.AddTask(task);
    std::string msg;
    if (!cluster->SubmitJob(jobspec, &msg)) {
        std::cerr << "Error in submit job: " << msg << std::endl;
    }
    std::cout << "Job ID = " << jobspec.job_id << std::endl<<std::endl;
    
    std::cout << "=================  Get Jobs =================" << std::endl;
    JobList jobs = cluster->QueryJobList();
    for (JobList::iterator it = jobs.begin(); it != jobs.end(); it++) {
        JobPtr j = *it;
        char name[64] = { 0 };
        std::snprintf(name, sizeof(name), "%-10s", j->GetSpec().job_name.c_str());
        char state[64] = { 0 };
        std::snprintf(state, sizeof(state), "%-10s", ToString(j->GetJobInfo().job_state));
        std::cout << j->GetSpec().job_id<<"\t" << name << "\t" << state
            << "\t\t" << j->GetJobInfo().submit_time << "\t\t" << j->GetJobInfo().progress << std::endl;
    }

    std::cout << std::endl;
    std::cout<< "=================  Wait Job =================" << std::endl;
    std::cout << " Wait job " << jobspec.job_id << " to finish..." << std::endl;
    JobPtr job = cluster->QueryJob(jobspec.job_id);
    JobInfo info = job->GetJobInfo();
    int progress = info.progress;
    std::cout << "  Progress = " << progress << "%" << std::endl;
    while (info.job_state == JobState::Queued || info.job_state == JobState::Executing) {
        ::Sleep(1000);
        job->UpdateJobInfo(info);
        if (info.progress != progress) {
            progress = info.progress;
            std::cout << "  Progress = " << progress << "%" << std::endl;
        }
    }
    std::cout << "  Job done!" << std::endl << std::endl;
    TaskInfoList tasks = job->GetTaskList();
    for (TaskInfoList::iterator it = tasks.begin(); it != tasks.end(); it++) {
        std::cout << it->task_name << "\t" << int(it->task_state) << "\t" << it->exec_node << "\t" << it->progress
            << "\t" << it->finish_time << "\tExitCode = "<<it->exit_code<<std::endl;
    }
    delete cluster;
    return 0;
}
