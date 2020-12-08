package util

import (
	"context"
	"os"
	"os/signal"
	// "github.com/apex/log"
)

// var logger = log.WithFields(log.Fields{
// 	"component": "util",
// })

func WaitSignals(ctx context.Context, signals ...os.Signal) bool {
	var sigs = make(chan os.Signal, 1)
	defer close(sigs)

	signal.Notify(sigs, signals...)
	defer signal.Stop(sigs)

	// This select statement blocks the main thread until we catch the signal
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != context.Canceled {
			// logger.WithError(err).Errorf("context is done")
		}
		return false
	case _ = <-sigs:
		// logger.Debugf("Catched signal: %+v", sig)
		return false
	}
}
