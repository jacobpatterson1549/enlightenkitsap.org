package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
)

//go:embed resources
var _siteFS embed.FS

const (
	resources = "resources"
	events    = "events"
	about     = "about"
	perm      = 0764
	kiloByte  = 1_000 * 1
	megaByte  = 1_000 * kiloByte
	kB50      = 50 * kiloByte
	mB10      = 10 * megaByte
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage of site generator:")
	fmt.Fprintln(os.Stderr, "flags")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Warning: Overwrites the previous site")
}

type Config struct {
	Dest        string
	OneResource bool
}

// delete this section when debugging
func main() {
	var cfg Config
	flag.StringVar(&cfg.Dest, "dest", "", "the location to save the site files to")
	flag.BoolVar(&cfg.OneResource, "one-resource", false, "show all videos and resources on one page")
	flag.Usage = usage
	flag.Parse()
	if len(cfg.Dest) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	// uncomment the line below and change the package name to "internal"
	// to debug the compilation of the site's web pages:
	// func (cfg Config) WriteSite() {

	if err := writeFiles(cfg.Dest, cfg.OneResource); err != nil {
		fmt.Fprintf(os.Stderr, "generating site: %v\n", err)
		os.Exit(1)
	}
}

func writeFiles(dest string, oneResource bool) error {
	s := Site{
		removeAll:   os.RemoveAll,
		OneResource: oneResource,
		mkdirAll:    func(path string) error { return os.MkdirAll(path, perm) },
		writeFile:   func(name string, data []byte) error { return os.WriteFile(name, data, perm) },
		isNotExist:  os.IsNotExist,
		fSys:        _siteFS,
		dest:        dest,
		Name:        "Enl!ghten",
		Description: "Kitsap Community Forum",
	}
	if err := s.cleanDest(); err != nil {
		return fmt.Errorf("cleaning destination directory: %w", err)
	}
	if err := s.addMain(); err != nil {
		return fmt.Errorf("main site pages: %w", err)
	}
	if err := s.addEvents(); err != nil {
		return fmt.Errorf("event pages: %w", err)
	}
	return nil
}
