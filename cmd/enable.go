// Copyright © 2018 Tim Aebi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
