#include "lightsched.h"

namespace lightsched {

std::string ToString(const LabelList& labels);
const char* ToString(JobState state);
const char* ToString(TaskState state);
const char* ToString(NodeState state);
JobState ToJobState(const char* state);
TaskState ToTaskState(const char* state);
NodeState ToMachineState(const char* state);

}
