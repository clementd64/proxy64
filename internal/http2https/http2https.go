package http2https

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

type server struct {
	acmeWebroot string
}

func Run(ctx context.Context, log *slog.Logger, env func(string) string) error {
	addr := env("HTTP2HTTPS_LISTEN")
	if addr == "" {
		log.InfoContext(ctx, "HTTP2HTTPS_LISTEN not set, skipping http2https service")
		return nil
	}

	srv := &http.Server{
		Addr: addr,
		Handler: &server{
			acmeWebroot: env("ACME_WEBROOT"),
		},
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

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.acmeWebroot != "" && r.Method == http.MethodGet {
		if p := path.Clean(r.URL.Path); strings.HasPrefix(p, "/.well-known/acme-challenge/") {
			http.ServeFile(w, r, path.Join(s.acmeWebroot, p))
			return
		}
	}

	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
