package cmd

import (
	"fmt"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/node"
	"github.com/spf13/cobra"
)

var serverAddr *string

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		nodesvc := node.NewNodeServer(*serverAddr)
		nodesvc.Run()
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)

	port := fmt.Sprintf("%d", constant.DefaultNodePort)
	serverAddr = nodeCmd.Flags().StringP("server", "s", "127.0.0.1:"+port, "Address and port of API Server")
}
