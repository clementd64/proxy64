package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/clementd64/proxy64/cmd"
)

func main() {
	ctx := context.Background()
	if err := cmd.Run(ctx, slog.Default(), os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
