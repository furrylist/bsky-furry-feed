package api

import (
	_ "embed"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:embed html/root.txt
var rootPage []byte

func rootHandler(log *slog.Logger) (string, http.Handler) {
	var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			log.Info("request to non-existent path", slog.Any("path", r.URL.Path))
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
			return
		}
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(rootPage)

	}

	return "/", otelhttp.NewHandler(h, "root")
}
