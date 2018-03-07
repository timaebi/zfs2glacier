package cmd

import (
	"github.com/spf13/cobra"
	"github.com/timaebi/zfs2glacier/bkp"
	log "github.com/sirupsen/logrus"
)

// command line argument for zfs path filter
var filter string

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "create a backup of all filesystems that are due",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetLevel(log.DebugLevel)
		b, err := bkp.NewBatch(filter)
		if err != nil {
			return err
		}
		err = b.Init()
		if err != nil {
			return err
		}
		log.Debug("New batch initialized")
		err = b.Run()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&filter, "filter", "f", "", "restrict volumes to backup")
}
