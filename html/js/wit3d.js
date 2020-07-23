var nodeTemplate

function initNodeList(result, nodesDIV, tmplName) {
    nodeTemplate = $(tmplName).detach()
    for(var n in result){
        node = nodeTemplate.clone(true)
        let id = "node"+result[n].name
        $(node).attr("id", id)
        $(node).find('div[id^="nodeNameIP"]').text(result[n].name+" ("+result[n].address+")")
        $(node).find('div[id^="nodeOSName"]').text(result[n].platform.name)
        var gpusNode = $(node).find('div[id^="nodeGPUs"]')
        var gpuCount = result[n].resources.gpu.cards
        for(var g=0; g<gpuCount; g++) {
            $(gpusNode).append($('<img src="img/gpu.png " height="21px " width="21px "></img>'))
        }
        $(node).css("visibility", "visible")
        $(nodesDIV).append(node)
    }
}
