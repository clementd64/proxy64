package http2https

import (
	"net/http"
	"strconv"
)

func Listen(port int) error {
	return http.ListenAndServe(":"+strconv.Itoa(port), http.HandlerFunc(handle))
}

func handle(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
