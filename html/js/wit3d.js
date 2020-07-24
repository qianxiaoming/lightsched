var nodeUpdateTimer = undefined
var jobUpdateTimer = undefined
var taskUpdateTimer = undefined

function updateNodeList(result, nodesDIV) {
    for(var n in result){
        var node = $("#node-"+result[n].name)
        var platform = result[n].platform.name.replace("Microsoft ","")
        var resource = result[n].resources.cpu.cores+" 核, "+result[n].resources.memory/1024+"G内存"
        if (node.length > 0) {
            $(node).find('div[id^="nodeNameIP"]').text(result[n].name+" ("+result[n].address+")  " + platform)
            $(node).find('div[id^="nodeResource"]').text(resource)
        } else {
            node = nodeTemplate.clone(true)
            $(node).attr("id", "node-" + result[n].name)
            var gpusNode = $(node).find('div[id^="nodeGPUs"]')
            var gpuCount = result[n].resources.gpu.cards
            for(var g=0; g<gpuCount; g++) {
                $(gpusNode).append($('<img src="img/gpu.png " height="21px " width="21px "></img>'))
            }
            $(node).css("visibility", "visible")
            $(nodesDIV).append(node)
        }
        $(node).find('div[id^="nodeNameIP"]').text(result[n].name+" ("+result[n].address+")  " + platform)
        $(node).find('div[id^="nodeResource"]').text(resource)
        if (result[n].state == 0) {
            $(node).find('img[id^="nodeStatus"]').attr("src", "img/online.png")
        } else if (result[n].state == 1) {
            $(node).find('img[id^="nodeStatus"]').attr("src", "img/offline.png")
        } else {
            $(node).find('img[id^="nodeStatus"]').attr("src", "img/unknown.png")
        }
    }
}

function setNodeTimer() {
    if (nodeUpdateTimer == undefined) {
        nodeUpdateTimer = setInterval(function(){
            $.get("../nodes", function(result){
                updateNodeList(result, "#nodesDIV", "#nodeTemplate")
            });
        }, 5000)
    }
}

var jobRowString = '<td> <a href="tasks.html?jobid={id}&jobname={name}&update={update}">{name}</a> </td> \
                    <td>{state}</td> \
                    <td>{tasks}</td> \
                    <td> \
                        <div class="progress" style="height:9px; width: 100%"> \
                            <div class="progress-bar {progress-status}" style="width:{progress}%"></div> \
                        </div> \
                    </td> \
                    <td>{start}</td> \
                    <td>{finish}</td> \
                    <td><img id="cancel-{id}" class="operation_img" data-toggle="tooltip" data-placement="top" title="取消执行" src="img/cancel.png" alt="取消" height="21px" width="21px"> \
                        <img id="delete-{id}" class="operation_img" data-toggle="tooltip" data-placement="top" title="删除记录" src="img/delete.png" alt="删除" height="21px" width="21px"> \
                    </td>'

function getJobState(v) {
    if (v == 0) return "等待"
    if (v == 1) return "执行"
    if (v == 2) return "挂起"
    if (v == 3) return "完成"
    if (v == 4) return "失败"
    if (v == 5) return "取消"
}

function isActiveJobState(v) {
    if (v >= 3) return "no"
    return "yes"
}

function updateJobList(result, jobTable) {
    for(var n in result){
        var tr = $("#job-"+result[n].id)
        if (tr.length == 0) {
            tr = $("<tr id=\"job-"+result[n].id+"\" state="+result[n].state+"></tr>")
            $(jobTable).prepend(tr)
        } else {
            var state = tr.attr("state")
            if (state == "3" || state=="4" || state=="5")
                continue
        }
        row = jobRowString.replace(/{id}/g, result[n].id)
        row = row.replace(/{name}/g, result[n].name)
        row = row.replace("{update}", isActiveJobState(result[n].state))
        row = row.replace("{state}", getJobState(result[n].state))
        row = row.replace("{tasks}", result[n].tasks)
        if (result[n].state == 4 || result[n].state == 5)
            row = row.replace("{progress-status}", "bg-danger")
        else
            row = row.replace("{progress-status}", "")
        row = row.replace("{progress}", result[n].progress)
        if (result[n].hasOwnProperty("exec_time"))
            row = row.replace("{start}", result[n].exec_time.substring(5, 19))
        else
            row = row.replace("{start}", "")
        if (result[n].hasOwnProperty("finish_time"))
            row = row.replace("{finish}", result[n].finish_time.substring(5, 19))
        else
            row = row.replace("{finish}", "")
        $(tr).html(row)

        $("#cancel-"+result[n].id).click(function(){
            var id = $(this).attr("id").substring(7, $(this).attr("id").length)
            $.ajax({
                url: "../jobs/"+id+"/_terminate",
                type: "PUT",
                success: function(result) {
                    $.get("../jobs", function(result){
                        updateJobList(result, "#job-table")
                    });
                }
            });
        });
        $("#delete-"+result[n].id).click(function(){
            var id = $(this).attr("id").substring(7, $(this).attr("id").length)
            $.ajax({
                url: "../jobs/"+id,
                type: "DELETE",
                success: function(result) {
                    $("#job-"+id).remove()
                }
            });
        });
    }
}

function setJobUpdateTimer() {
    if (jobUpdateTimer == undefined) {
        jobUpdateTimer = setInterval(function(){
            $.get("../jobs", function(result){
                updateJobList(result, "#job-table")
            });
        }, 2000)
    }
}

function getQueryString(name) {
    var reg = new RegExp("(^|&)" + name + "=([^&]*)(&|$)", "i");
    var r = window.location.search.substr(1).match(reg);
    if(r != null) return decodeURI(r[2]);
    return null;
}

var taskRowString ='<td>{number}</td><td>{name}</td> \
                    <td><img src="img/{state}.png" data-toggle="tooltip" title="{state-tip}" height="24px" width="24px"></td> \
                    <td>{node}</td> \
                    <td><div class="progress" style="height:9px; width: 100%"> \
                            <div class="progress-bar {progress-status}" style="width:{progress}%"></div> \
                        </div> \
                    </td> \
                    <td>{start}</td> \
                    <td>{finish}</td> \
                    <td>{error}</td> \
                    <td><button type="button" class="btn btn-primary btn-sm" data-toggle="modal" data-target="#taskLogModal" data-id="{id}">查看日志</button></td>'

function getTaskState(v) {
    if (v == 0) return "queued"
    if (v == 1) return "scheduled"
    if (v == 2) return "dispatching"
    if (v == 3) return "executing"
    if (v == 4) return "completed"
    if (v == 5) return "failed"
    if (v == 6) return "aborted"
    if (v == 7) return "terminated"
}

function getTaskStateTip(v) {
    if (v == 0) return "等待"
    if (v == 1) return "已调度"
    if (v == 2) return "分发中"
    if (v == 3) return "执行中"
    if (v == 4) return "已完成"
    if (v == 5) return "失败"
    if (v == 6) return "异常中止"
    if (v == 7) return "已取消"
}

function updateTaskList(result, taskTable) {
    for(var n in result){
        var taskid = result[n].id.replace(".", "-")
        taskid = taskid.replace(".", "-")
        var tr = $("#task-"+taskid)
        if (tr.length == 0) {
            tr = $("<tr id=\"task-"+taskid+"\" state="+result[n].state+"></tr>")
            $(taskTable).append(tr)
        } else {
            var state = tr.attr("state")
            if (state=="4" || state=="5" || state == "6" || state == "7")
                continue
        }
        row = taskRowString.replace(/{id}/g, result[n].id)
        row = row.replace("{name}", result[n].name)
        row = row.replace("{number}", parseInt(n)+1)
        row = row.replace("{state}", getTaskState(result[n].state))
        row = row.replace("{state-tip}", getTaskStateTip(result[n].state))
        if (result[n].hasOwnProperty("node"))
            row = row.replace("{node}", result[n].node)
        else
            row = row.replace("{node}", "")
        if (result[n].state == 5 || result[n].state == 6 || result[n].state == 7)
            row = row.replace("{progress-status}", "bg-danger")
        else
            row = row.replace("{progress-status}", "")
        row = row.replace("{progress}", result[n].progress)
        if (result[n].hasOwnProperty("start_time"))
            row = row.replace("{start}", result[n].start_time.substring(5, 19))
        else
            row = row.replace("{start}", "")
        if (result[n].hasOwnProperty("finish_time"))
            row = row.replace("{finish}", result[n].finish_time.substring(5, 19))
        else
            row = row.replace("{finish}", "")
        if (result[n].hasOwnProperty("error"))
            row = row.replace("{error}", "<img src=\"img/alert.png\" data-toggle=\"tooltip\" title=\""+result[n].error+"\" height=\"18px\" width=\"18px\">")
        else
            row = row.replace("{error}", "")
        $(tr).html(row)

        $('#taskLogModal').on('show.bs.modal', function (event) {
            var taskid = $(event.relatedTarget).data('id')
            var modal = $(this)
            modal.find('#taskLogViewer').load("../tasks/"+taskid+"/log")
        })
    }
}

function setTaskUpdateTimer(jobid) {
    if (taskUpdateTimer == undefined) {
        taskUpdateTimer = setInterval(function(){
            $.get("../tasks?jobid="+jobid, function(result){
                updateTaskList(result, "#task-table")
            });
        }, 2000)
    }
}