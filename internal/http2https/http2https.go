package http2https

import (
	"net/http"
)

func Listen(addr string) error {
	return http.ListenAndServe(addr, http.HandlerFunc(handle))
}

func handle(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
