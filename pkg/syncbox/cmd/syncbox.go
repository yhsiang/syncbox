package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yhsiang/syncbox/pkg/syncbox"
)

var (
	serverCmd = &cobra.Command{
		Use:   "syncbox",
		Short: "syncbox is a dropbox-like client",
		Long:  `syncbox is dropbox-like client to sync your files to syncbox server.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return errors.New("path could not be empty.")
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			watcher := syncbox.NewFileWatcher(ctx, args[0])
			watcher.OnChange(func(files []syncbox.File) {
				for _, file := range files {
					fmt.Printf("%+v\n", file)
				}
				fmt.Println("=========")
			})

			err := watcher.Run()
			if err != nil {
				return err
			}

			return nil
		},
	}
)

// Execute executes the root command.
func ExecuteClientCmd() error {
	serverCmd.SetUsageTemplate(`syncbox [directory path] e.g., synbox /tmp/dropbox/server`)
	return serverCmd.Execute()
}

func init() {
}
