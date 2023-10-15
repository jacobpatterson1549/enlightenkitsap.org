package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed build/site
var _siteFS embed.FS

//go:generate go run enlightenkitsap.org/internal -dest=build/site
func main() {
	// uncomment the line below to debug compilation of the site:
	// internal.WriteSite("build/site")

	cfg := new(config)
	if err := cfg.parseArgsAndEnv(os.Stdout, os.Args...); err != nil {
		log.Fatalf("parsing program options: %v", err)
	}
	h, err := newHandler(_siteFS)
	if err != nil {
		log.Fatalf("creating site page handler: %v", err)
	}
	addr := ":" + cfg.port
	log.Println("Serving site at http://127.0.0.1" + addr)
	log.Println("Press Ctrl-C to stop")
	http.ListenAndServe(addr, h)
}

func newHandler(siteFS fs.FS) (http.Handler, error) {
	subFS, err := fs.Sub(siteFS, "build/site")
	if err != nil {
		return nil, fmt.Errorf("getting siteFS: %w", err)
	}
	hfs := http.FS(subFS)
	h := http.FileServer(hfs)
	h = withProxy(h, "/", "/home.html")
	h = withBasicCacheControl(h)
	h = withContentEncoding(h)
	return h, nil
}
