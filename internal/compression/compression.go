package compression

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
)

var allowedCompressTypes = []string{"text/html", "application/json"}

type gzipResponseWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewCompressWriter(w http.ResponseWriter) *gzipResponseWriter {
	return &gzipResponseWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *gzipResponseWriter) Header() http.Header {
	return c.w.Header()
}

func (c *gzipResponseWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *gzipResponseWriter) WriteHeader(statusCode int) {
	c.w.Header().Set("Content-Encoding", "gzip")
	c.w.WriteHeader(statusCode)
}

func (c *gzipResponseWriter) Close() error {
	return c.zw.Close()
}

type gzipReadCloser struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func NewCompressReader(r io.ReadCloser) (*gzipReadCloser, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReadCloser{
		r:  r,
		zr: zr,
	}, nil
}

func (c *gzipReadCloser) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *gzipReadCloser) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func GzipCompressDecompress(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ow := w
		supportsContentType := false

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if contentType := r.Header.Get("Content-Type"); contentType != "" {
			supportsContentType = slices.Contains(allowedCompressTypes, contentType)
		}
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip && (r.Method == http.MethodGet || supportsContentType) {
			cw := NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
