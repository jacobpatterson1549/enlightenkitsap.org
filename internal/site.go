package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"path"
	"text/template"
)

//go:embed resources
var siteFS embed.FS

const (
	resources = "resources"
	perm      = 0764
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage of site generator:")
	fmt.Fprintln(os.Stderr, "flags")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "Warning: Overwrites the previous site")
}

func main() {
	// uncomment the line below and change the package name to "internal"
	// to debug the compilation of the site's web pages:
	// func WriteSite(dest string) {

	var dest string
	flag.StringVar(&dest, "dest", "", "the location to save the site files to")
	flag.Usage = usage
	flag.Parse()

	if len(dest) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := writeFiles(dest); err != nil {
		fmt.Fprintf(os.Stderr, "generating site: %v\n", err)
		os.Exit(1)
	}
}

func writeFiles(dest string) error {
	if err := cleanDest(dest); err != nil {
		return fmt.Errorf("cleaning destination directory: %w", err)
	}

	site := Site{
		Name:        "Enl!ghten",
		Description: "Kitsap Community Forum",
	}
	pages := []struct {
		srcDir   string
		fileName string
		name     string
		data     interface{} // TODO: not used, add additional data
	}{
		{"", "home", "Home Page", nil},
		{"about", "board-members", "Board Members", nil},
		{"about", "contact-us", "Contact Us", nil},
		{"about", "donations", "Donations", nil},
		{"about", "location", "Where Are We Located?", nil},
		{"about", "mission-statement", "Mission Statement", nil},
		{"about", "purpose-statement", "Purpose Statement", nil},
		{"about", "volunteers", "Volunteers", nil},
		{"events", "calendar", "Calendar", nil},
		{"events", "future-events", "Future Events", nil},
		{"events", "past-events", "Past Events", nil},
		{"events", "sign-up", "Sign Up For Events", nil},
	}
	for _, pg := range pages {
		srcDir := path.Join(resources, pg.srcDir)
		fileName := pg.fileName + ".html"
		p := Page{
			Name: pg.name,
			Data: pg.data,
		}
		data := Data{
			Site: site,
			Page: p,
		}
		if err := writeFile(dest, srcDir, fileName, data); err != nil {
			return fmt.Errorf("writing file %v, %w", fileName, err)
		}
	}
	// TODO: write non-files from resources
	return nil
}

func cleanDest(dest string) error {
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing old version of site: %w", err)
	}
	if err := os.MkdirAll(dest, perm); err != nil {
		return fmt.Errorf("creating new site directory: %w", err)
	}
	return nil
}

func writeFile(destDir, srcDir, name string, data interface{}) error {
	src := path.Join(srcDir, name)
	t, err := lookupTemplate(src)
	if err != nil {
		return fmt.Errorf("looking up template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	b := buf.Bytes()
	dest := path.Join(destDir, name)
	if err := os.WriteFile(dest, b, perm); err != nil {
		return fmt.Errorf("writing template: %w", err)
	}
	return nil
}

func lookupTemplate(content string) (*template.Template, error) {
	mainHTML := path.Join(resources, "main.html")
	mainCSS := path.Join(resources, "index.css")
	navHTML := path.Join(resources, "nav.html")
	navCSS := path.Join(resources, "nav.css")
	t, err := template.New("main.html").
		Option("missingkey=error").
		ParseFS(siteFS, mainHTML, mainCSS, navHTML, navCSS, content)
	if err != nil {
		return nil, fmt.Errorf("parsing template filesystem: %w", err)
	}
	return t, nil
}

type (
	Data struct {
		Site Site
		Page Page
	}
	Site struct {
		Name        string
		Description string
	}
	Page struct {
		Name string
		Data interface{}
	}
)
