package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
)

// statusCmd represents the status command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print current version",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
