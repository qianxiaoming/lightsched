#include <iostream>
#include <lightsched.h>

int main()
{
    using namespace lightsched;
    ComputingCluster* cluster = new ComputingCluster("127.0.0.1");
    if (!cluster->IsConnected()) {
        std::cout << "Cannot connect to server" << std::endl;
        delete cluster;
        return 1;
    } else
        std::cout << "Cluster " << cluster->GetName() << " connected" << std::endl;

    JobSpec jobspec("Hello", "c:/windows/my.exe");
    jobspec.environments = "LD_LIBRARY_PATH=/tmp";
    jobspec.labels["app"] = "test";
    jobspec.work_dir = "c:/temp";
    cluster->SubmitJob(jobspec);
    delete cluster;
    return 0;
}
