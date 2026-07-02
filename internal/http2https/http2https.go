package http2https

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

func Listen(ctx context.Context, log *slog.Logger, env func(string) string) error {
	addr := env("HTTP2HTTPS_LISTEN")
	if addr == "" {
		log.InfoContext(ctx, "HTTP2HTTPS_LISTEN not set, skipping http2https service")
		return nil
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handle),
	}

	go func() {
		<-ctx.Done()
		log.InfoContext(ctx, "Shutting down server")
		srv.Shutdown(context.Background())
	}()

	log.InfoContext(ctx, "Starting server", "addr", addr)
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func handle(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
