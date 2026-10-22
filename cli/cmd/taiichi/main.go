// Taiichi is the command-line client for the Taiichi server.
package main

import (
	"context"
	"errors"
	"os"

	"charm.land/fang/v2"

	"github.com/ngscheurich/taiichi/cli/internal/command"
)

func main() {
	os.Exit(run())
}

func run() int {
	err := fang.Execute(context.Background(), command.NewRoot(),
		fang.WithNotifySignal(os.Interrupt),
		fang.WithoutManpage(),
	)
	switch {
	case err == nil:
		return 0
	case errors.Is(err, context.Canceled):
		return 130
	default:
		return 1
	}
}
