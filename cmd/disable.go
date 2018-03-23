package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timaebi/zfs2glacier/bkp"
	"github.com/timaebi/go-zfs"
)

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:   "disable [filesystem]",
	Short: "disable backup for a filesystem",
	Long:  `This will not delete any files in the aws cloud.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ds, err := zfs.GetDataset(args[0])
		check(err)
		err = ds.SetProperty(bkp.BackupEnabled, "false")
		check(err)
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
