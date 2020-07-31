package relaybaton

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "relaybaton",
	Short: "TODO",
	Long:  "TODO",
}

func init() {
	RootCmd.PersistentFlags().String("config", "", "TODO")
	RootCmd.AddCommand(ClientCmd)
	RootCmd.AddCommand(ServerCmd)
}
