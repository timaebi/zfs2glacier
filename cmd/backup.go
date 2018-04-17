package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timaebi/zfs2glacier/bkp"
)

// command line argument for zfs path filter
var filter string

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "create a backup of all filesystems that are due",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bkp.NewBatch(filter)
		check(err)
		err = b.Init()
		check(err)
		err = b.Run()
		check(err)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&filter, "filter", "f", "", "restrict volumes to backup")
}
