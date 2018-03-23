package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timaebi/zfs2glacier/bkp"
	"fmt"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "print current backup status to stdout",
	Long:  `Shows all zfs filesystems for which a backup should be created. It also shows the last backup status and date`,
	Run: func(cmd *cobra.Command, args []string) {
		batch, err := bkp.NewBatch(filter)
		if err != nil {
			fmt.Print(err)
			return
		}
		err = batch.Init()
		if err != nil {
			fmt.Print(err)
			return
		}
		batch.Print()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVarP(&filter, "filter", "f", "", "restrict volumes to backup")
}
