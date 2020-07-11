package cmd

import (
	"fmt"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/node"
	"github.com/spf13/cobra"
)

var (
	serverAddr *string
	myhostname *string
	cpuSetting *string
	gpuSetting *string
	memSetting *string
	labSetting *string
)

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run as Node Server",
	Long: `Run as a Node Server of the cluster specified by API Server address. Node Server 
can accept tasks which are scheduled by API Server.`,
	Run: func(cmd *cobra.Command, args []string) {
		nodesvc := node.NewNodeServer(*serverAddr, *myhostname)
		if nodesvc == nil {
			return
		}
		nodesvc.Run(*cpuSetting, *gpuSetting, *memSetting, *labSetting)
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)

	port := fmt.Sprintf("%d", constant.DefaultNodePort)
	serverAddr = nodeCmd.Flags().StringP("server", "s", "127.0.0.1:"+port, "Address and port of API Server")
	myhostname = nodeCmd.Flags().StringP("name", "n", "", "Host name of this machine")
	cpuSetting = nodeCmd.Flags().StringP("cpu", "c", "", "Setting string for CPU resource: \"cores=16;freq=2400\"")
	gpuSetting = nodeCmd.Flags().StringP("gpu", "g", "", "Setting string for GPU resource: \"cards=1;cores=3865;mem=11;cuda=1020\"")
	memSetting = nodeCmd.Flags().StringP("mem", "m", "", "Setting string for memory resource: \"64000\"")
	labSetting = nodeCmd.Flags().StringP("labels", "l", "", "Setting string for node lables: \"key1=value1;key2=value2\"")
}
