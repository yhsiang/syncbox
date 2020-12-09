package cmd

import (
	"context"
	"fmt"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yhsiang/syncbox/pkg/syncbox"
	"github.com/yhsiang/syncbox/pkg/util"
)

var serverUrl = fmt.Sprintf("ws://%s/", ServerAddr)

var (
	clientCmd = &cobra.Command{
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

			fileWatcher := syncbox.NewFileWatcher(ctx, args[0])
			client := syncbox.NewSyncClient(serverUrl, fileWatcher)
			fileWatcher.OnChange(client.EmitFileChange)

			client.Connect(ctx)
			defer client.Disconnect()

			go fileWatcher.Run()

			util.WaitSignals(ctx, syscall.SIGINT, syscall.SIGTERM)
			return nil
		},
	}
)

// Execute executes the root command.
func ExecuteClientCmd() error {
	clientCmd.SetUsageTemplate(`syncbox [directory path] e.g., synbox /tmp/dropbox/server`)
	return clientCmd.Execute()
}

func init() {
}
