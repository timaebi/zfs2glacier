package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timaebi/go-zfs"
	"github.com/timaebi/zfs2glacier/bkp"
	"strconv"
)

var incrementalInterval uint64

// enableCmd represents the enable command
var enableCmd = &cobra.Command{
	Use:   "enable [filesystem]",
	Short: "enable automatic backups for the given volume",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ds, err := zfs.GetDataset(args[0])
		check(err)
		err = ds.SetProperty(bkp.BackupEnabled, "true")
		check(err)
		err = ds.SetProperty(bkp.IncrementalInterval, strconv.FormatUint(incrementalInterval, 10))
		check(err)
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
	enableCmd.Flags().Uint64VarP(&incrementalInterval, "incremental", "i", 2592000, "time between two incremental backups, defaults to 30 days")
}
