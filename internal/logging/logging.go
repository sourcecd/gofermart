package logging

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

var Slog *slog.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func WriteLogging(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		encCheck := r.Header.Get("Accept-Encoding")

		h(&lw, r)

		duration := time.Since(start)
		Slog.Info(
			"request",
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.Duration("duration", duration),
			slog.Int("StatusCode", responseData.status),
			slog.Int("content-length", responseData.size),
			slog.String("Accept-Encoding", encCheck),
		)
	}
}
