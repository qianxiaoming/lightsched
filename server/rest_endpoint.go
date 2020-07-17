package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qianxiaoming/lightsched/model"
)

func responseError(code int, format string, err error, c *gin.Context) {
	str := fmt.Sprintf(format, err)
	log.Printf(str)
	if code != 0 {
		c.String(code, str)
	} else {
		c.Writer.WriteString(str)
	}
}

// JobEndpoint 是Job资源对象的RESTful API实现接口
type JobEndpoint struct{}

func (e JobEndpoint) registerRoute() {
	// state=Executing&sort=submit/state&offset=0&limit=15
	apiserver.restRouter.GET(e.restPrefix(), e.getJobs)
	apiserver.restRouter.GET(e.restPrefix()+"/:id", e.getJob)
	apiserver.restRouter.POST(e.restPrefix(), e.createJob)
	apiserver.restRouter.GET(e.restPrefix()+"/:id/_terminate", e.terminateJob)
	apiserver.restRouter.DELETE(e.restPrefix()+"/:id", e.deleteJob)
}

func (e JobEndpoint) restPrefix() string {
	return "/jobs"
}

func (e JobEndpoint) getJobs(c *gin.Context) {
	var filterState *model.JobState = nil
	if v := c.Query("state"); len(v) > 0 {
		state := model.JobStateFromString(v)
		filterState = &state
	}
	sortField := model.SortJobByDefault
	if v := c.Query("sort"); len(v) > 0 {
		if v == "state" {
			sortField = model.SortJobByState
		} else if v == "submit" {
			sortField = model.SortJobBySubmit
		}
	}
	offset := 0
	if v := c.Query("offset"); len(v) > 0 {
		offset, _ = strconv.Atoi(v)
	}
	limits := -1
	if v := c.Query("limits"); len(v) > 0 {
		limits, _ = strconv.Atoi(v)
	}

	allJobs := apiserver.requestListJobs(filterState, sortField, offset, limits)
	if allJobs != nil {
		c.JSON(http.StatusOK, allJobs)
	} else {
		c.Status(http.StatusNotFound)
	}
}

func (e JobEndpoint) getJob(c *gin.Context) {
	jobid := c.Params.ByName("id")
	jobInfo := apiserver.requestGetJob(jobid)
	if jobInfo == nil {
		c.Status(http.StatusNotFound)
	} else {
		c.JSON(http.StatusOK, jobInfo)
	}
}

func (e JobEndpoint) createJob(c *gin.Context) {
	spec := &model.JobSpec{}
	if err := c.BindJSON(spec); err == nil {
		log.Printf("Request to create job \"%s\"(%s) in queue \"%s\" with %d task group(s)...\n", spec.Name, spec.ID, spec.Queue, len(spec.GroupSpecs))
		err = apiserver.requestCreateJob(spec)
		if err == nil {
			c.JSON(http.StatusCreated, gin.H{"id": spec.ID})
		} else {
			responseError(http.StatusBadRequest, "Create job failed: %v", err, c)
		}
	} else {
		responseError(http.StatusBadRequest, "Parse request failed: %v", err, c)
	}
}

func (e JobEndpoint) deleteJob(c *gin.Context) {
	id := c.Params.ByName("id")
	c.JSON(http.StatusOK, gin.H{"job": id})
}

func (e JobEndpoint) terminateJob(c *gin.Context) {
	id := c.Params.ByName("id")
	if err := apiserver.requestTerminateJob(id); err != nil {
		responseError(http.StatusBadRequest, "Unable to terminate job: %v", err, c)
		return
	}
	c.Status(http.StatusAccepted)
}

// TaskEndpoint 是Task资源对象的RESTful API实现接口
type TaskEndpoint struct{}

func (e TaskEndpoint) registerRoute() {
	apiserver.restRouter.GET(e.restPrefix(), func(c *gin.Context) {
		jobid := c.Query("jobid")
		tasks := apiserver.requestGetJobTasks(jobid)
		if tasks == nil {
			c.Status(http.StatusNotFound)
		} else {
			c.JSON(http.StatusOK, tasks)
		}
	})
	apiserver.restRouter.GET(e.restPrefix()+"/:id", func(c *gin.Context) {
		taskid := c.Params.ByName("id")
		taskInfo := apiserver.requestGetTask(taskid)
		if taskInfo == nil {
			c.Status(http.StatusNotFound)
		} else {
			c.JSON(http.StatusOK, taskInfo)
		}
	})
}

func (e TaskEndpoint) restPrefix() string {
	return "/tasks"
}

// QueueEndpoint 是Queue资源对象的RESTful API实现接口
type QueueEndpoint struct{}

func (e QueueEndpoint) registerRoute() {
	apiserver.restRouter.GET(e.restPrefix(), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"tasks": [...]string{"t001", "t002", "t003"},
		})
	})
}

func (e QueueEndpoint) restPrefix() string {
	return "/queues"
}

// NodeEndpoint 是Node资源对象的RESTful API实现接口
type NodeEndpoint struct{}

func (e NodeEndpoint) registerRoute() {
	apiserver.restRouter.GET(e.restPrefix(), func(c *gin.Context) {
		allNodes := apiserver.requestListNodes()
		if allNodes != nil {
			c.JSON(http.StatusOK, allNodes)
		} else {
			c.Status(http.StatusNotFound)
		}
	})
}

func (e NodeEndpoint) restPrefix() string {
	return "/nodes"
}
