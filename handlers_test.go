package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithProxy(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"/a", "/a"},
		{"/replace", "/redirect"},
		{"/redirect", "/redirect"},
	}
	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			h1 := func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(r.URL.Path))
			}
			h2 := withProxy(http.HandlerFunc(h1), "/replace", "/redirect")
			r := httptest.NewRequest("", test.url, nil)
			w := httptest.NewRecorder()
			h2.ServeHTTP(w, r)
			if want, got := test.want, w.Body.String(); got != want {
				t.Fatalf("wanted body to be %q, got %q", want, got)
			}
		})
	}
}

func TestWithCacheControl(t *testing.T) {
	msg := "OK_1549"
	h1 := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(msg))
	}
	h2 := withCacheControl(http.HandlerFunc(h1), time.Minute)
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()
	h2.ServeHTTP(w, r)
	if want, got := w.Body.String(), msg; want != got {
		t.Fatalf("wanted body to be %q, got %q", want, got)
	}
	gotHeader := w.Header()
	if want, got := "max-age=60", gotHeader.Get("Cache-Control"); want != got {
		t.Errorf("missing max-age Cache-Control header: got: %q", got)
	}
}

func TestWithBasicCacheControl(t *testing.T) {
	msg := "once"
	h1 := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(msg))
	}
	h2 := withBasicCacheControl(http.HandlerFunc(h1))
	r1 := httptest.NewRequest("", "/", nil)
	w1 := httptest.NewRecorder()
	h2.ServeHTTP(w1, r1)
	r2 := httptest.NewRequest("", "/", nil)
	w2 := httptest.NewRecorder()
	h2.ServeHTTP(w2, r2)
	header2 := w2.Header()
	v := header2.Values("Cache-Control")
	if want, got := 1, len(v); want != got {
		t.Errorf("header should only be added %vx, got %vx", want, got)
	}
}

func TestWithContentEncoding(t *testing.T) {
	msg := "OK_gzip"
	tests := []struct {
		name    string
		ae      string
		wantCE  string
		getBody func(t *testing.T, r io.Reader) io.Reader
	}{
		{
			name:   "gzip",
			ae:     "gzip, deflate, br",
			wantCE: "gzip",
			getBody: func(t *testing.T, r io.Reader) io.Reader {
				t.Helper()
				gr, err := gzip.NewReader(r)
				if err != nil {
					t.Fatalf("creating gzip reader: %v", err)
				}
				return gr
			},
		},
		{
			name:   "UNKNOWN",
			ae:     "UNKNOWN",
			wantCE: "",
			getBody: func(t *testing.T, r io.Reader) io.Reader {
				return r
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h1 := func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(msg))
			}
			h2 := withContentEncoding(http.HandlerFunc(h1))
			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/", nil)
			r.Header.Add("Accept-Encoding", test.ae)
			h2.ServeHTTP(w, r)
			gotHeader := w.Header()
			gotCE := gotHeader.Get("Content-Encoding")
			if test.wantCE != gotCE {
				t.Fatalf("wanted %q Content-Encoding, got: %q",
					test.wantCE, gotCE)
			}
			gr := test.getBody(t, w.Body)
			b, err := io.ReadAll(gr)
			if err != nil {
				t.Fatalf("reading gzip encoded message: %v", err)
			}
			if want, got := msg, string(b); want != got {
				t.Errorf("body not encoded as desired: wanted %q, got %q",
					want, got)
			}
		})
	}
}
