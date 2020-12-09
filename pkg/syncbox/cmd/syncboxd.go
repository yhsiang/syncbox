package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yhsiang/syncbox/pkg/syncbox"
)

var ServerAddr = "localhost:3000"

var (
	serverCmd = &cobra.Command{
		Use:   "syncboxd",
		Short: "syncboxd is a dropbox-like server",
		Long:  `syncboxd is dropbox-like server to sync your files from syncbox client.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return errors.New("path could not be empty.")
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fileWatcher := syncbox.NewFileWatcher(ctx, args[0])
			server := syncbox.NewServer(ctx, ServerAddr, fileWatcher)

			go fileWatcher.Run()

			fmt.Printf("server listen on %s\n", ServerAddr)
			return server.ListenAndServe()
		},
	}
)

// Execute executes the root command.
func ExecuteServerCmd() error {
	serverCmd.SetUsageTemplate(`syncboxd [directory path] e.g., synbox /tmp/dropbox/server`)
	return serverCmd.Execute()
}

func init() {
}
