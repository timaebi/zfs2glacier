package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	"log/syslog"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zfs2glacier",
	Short: "backup zfs filesystems to aws glacier",
	Long:  ``,
	PreRun: func(cmd *cobra.Command, args []string) {
		hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_USER, "")
		if err != nil {
			panic(err)
		}
		log.AddHook(hook)
		if verbose {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging to stdout")
}

func check(err error) {
	if err != nil {
		log.Error(err)
		log.Exit(1)
	}
}
