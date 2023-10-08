package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed build/site
var _siteFS embed.FS

//go:generate go run enlightenkitsap.org/internal -dest=build/site
func main() {
	// uncomment the line below to debug compilation of the site:
	// internal.WriteSite("build/site")

	addr := parsePort()
	h := handler(_siteFS)
	log.Println("Serving site at http://127.0.0.1" + addr)
	log.Println("Press Ctrl-C to stop")
	http.ListenAndServe(addr, h)
}

func parsePort() string {
	args := os.Args
	programName, programArgs := args[0], args[1:]
	flagSet := flag.NewFlagSet(programName, flag.ExitOnError)
	port := flagSet.String("port", "8000", "the port to run the site on")
	if err := parseFlagSet(flagSet, programArgs...); err != nil {
		log.Fatalf("parsing program flags: %v", err)
	}
	return ":" + *port
}

func handler(siteFS fs.FS) http.Handler {
	subFS, err := fs.Sub(siteFS, "build/site")
	if err != nil {
		log.Fatalf("getting siteFS: %v", err)
	}
	hfs := http.FS(subFS)
	h := http.FileServer(hfs)
	h = withProxy(h, "/", "/home.html")
	h = withBasicCacheControl(h)
	h = withContentEncoding(h)
	return h
}

// parseFlagSet parses the FlagSet and overlays environment flags.
// Flags that match environment variables with an uppercase version of their
// names, with underscores instead of hyphens are overwritten.
func parseFlagSet(fs *flag.FlagSet, programArgs ...string) error {
	if err := fs.Parse(programArgs); err != nil {
		return fmt.Errorf("parsing program args: %w", err)
	}
	if err := parseEnvVars(fs); err != nil {
		return fmt.Errorf("setting value from environment variable: %w", err)
	}
	return nil
}

func parseEnvVars(fs *flag.FlagSet) error {
	var lastErr error
	fs.VisitAll(func(f *flag.Flag) {
		upperName := strings.ToUpper(f.Name)
		name := strings.ReplaceAll(upperName, "-", "_")
		val, ok := os.LookupEnv(name)
		if !ok {
			return
		}
		if err := f.Value.Set(val); err != nil {
			lastErr = err
		}
	})
	return lastErr
}
