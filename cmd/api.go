package cmd

import (
	"os"
	"path/filepath"

	"github.com/qianxiaoming/lightsched/constant"
	"github.com/qianxiaoming/lightsched/server"
	"github.com/spf13/cobra"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Run as API Server",
	Long: `Run as a API Server of the cluster. API Server accepts jobs submitted by clients and 
	schedule them to work nodes to execute.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := os.Executable()
		if err != nil {
			panic(err)
		}
		confPath := filepath.Join(filepath.Dir(path), constant.APISeverConfigFile)
		apisvc := server.NewAPIServer(confPath)
		apisvc.Run()
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// apiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// apiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
