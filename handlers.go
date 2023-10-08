package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

func withProxy(h http.Handler, src, dest string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == src {
			r.URL.Path = dest
		}
		h.ServeHTTP(w, r)
	}
}

func withBasicCacheControl(h http.Handler) http.HandlerFunc {
	day := 24 * time.Hour
	year := 365 * day
	return func(w http.ResponseWriter, r *http.Request) {
		ext := path.Ext(r.URL.Path)
		h2 := withCacheControl(h, year)
		switch ext {
		case ".html", "":
			h2 = withCacheControl(h, day)
		}
		h2.ServeHTTP(w, r)
	}
}

func withCacheControl(h http.Handler, d time.Duration) http.HandlerFunc {
	maxAge := "max-age=" + strconv.Itoa(int(d.Seconds()))
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", maxAge)
		h.ServeHTTP(w, r)
	}
}

func withContentEncoding(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := r.Header.Get("Accept-Encoding")
		if strings.Contains(enc, "gzip") {
			gzw := gzip.NewWriter(w)
			defer gzw.Close()
			wrw := wrappedResponseWriter{
				Writer:         gzw,
				ResponseWriter: w,
			}
			wrw.Header().Set("Content-Encoding", "gzip")
			w = wrw
		}
		h.ServeHTTP(w, r)
	}
}

type wrappedResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (wrw wrappedResponseWriter) Write(p []byte) (n int, err error) {
	return wrw.Writer.Write(p)
}
