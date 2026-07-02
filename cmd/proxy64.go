package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/clementd64/proxy64/internal/http2https"
	"github.com/clementd64/proxy64/internal/nat64"
	"github.com/clementd64/proxy64/internal/sni"
)

func Run(ctx context.Context, log *slog.Logger, env func(string) string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	var wg sync.WaitGroup

	for name, run := range map[string]func(context.Context, *slog.Logger, func(string) string) error{
		"http2https": http2https.Run,
		"nat64":      nat64.Run,
		"sni":        sni.Run,
	} {
		wg.Go(func() {
			log := log.With("svc", name)
			if err := run(ctx, log, env); err != nil {
				log.ErrorContext(ctx, "service stopped with error", "error", err)
				cancel(err)
			}
		})
	}

	wg.Wait()
	log.InfoContext(ctx, "all services stopped")
	return context.Cause(ctx)
}
