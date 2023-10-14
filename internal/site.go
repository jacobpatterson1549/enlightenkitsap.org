package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
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

// delete this section when debugging
func main() {
	var dest string
	flag.StringVar(&dest, "dest", "", "the location to save the site files to")
	flag.Usage = usage
	flag.Parse()

	// uncomment the line below and change the package name to "internal"
	// to debug the compilation of the site's web pages:
	// func WriteSite(dest string) {

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
	s := Site{
		dest:        dest,
		Name:        "Enl!ghten",
		Description: "Kitsap Community Forum",
	}
	if err := s.addMain(); err != nil {
		return fmt.Errorf("main site pages: %w", err)
	}
	if err := s.addEvents(); err != nil {
		return fmt.Errorf("event pages: %w", err)
	}
	return nil
}

func (s *Site) addMain() error {
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
		{"events", "sign-up", "Sign Up For Events", nil},
	}
	for _, pg := range pages {
		if err := s.addPage(pg.name, pg.srcDir, pg.fileName+".html", pg.data); err != nil {
			return fmt.Errorf("writing page: %w", err)
		}
	}
	imageDirs := []struct {
		src  string
		dest string
	}{
		{"", ""}, // root images from resources
		{"about", "board"},
	}
	for _, img := range imageDirs {
		src := path.Join(resources, img.src, "images")
		dest := path.Join(s.dest, "images", img.dest)
		if err := addImages(src, dest); err != nil {
			return fmt.Errorf("adding images from: %w", err)
		}
	}
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

func (s *Site) writeFile(srcDir, name string, data interface{}) error {
	if err := os.MkdirAll(s.dest, perm); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	src := path.Join(resources, srcDir, name)
	t, err := lookupMainTemplate(src)
	if err != nil {
		return fmt.Errorf("looking up template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	b := buf.Bytes()
	dest := path.Join(s.dest, name)
	if err := os.WriteFile(dest, b, perm); err != nil {
		return fmt.Errorf("writing template: %w", err)
	}
	return nil
}

func addImages(src, dest string) error {
	entries, err := siteFS.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading image directory: %w", err)
	}
	if err := os.MkdirAll(dest, perm); err != nil {
		return fmt.Errorf("creating image directory: %w", err)
	}
	for _, f := range entries {
		if err := addImage(f, src, dest); err != nil {
			return fmt.Errorf("adding image: %w", err)
		}
	}
	return nil
}

func addImage(f fs.DirEntry, src, dest string) error {
	if f.IsDir() {
		return fmt.Errorf("will not read directory from image folder")
	}
	n := f.Name()
	srcP := path.Join(src, n)
	b, err := siteFS.ReadFile(srcP)
	if err != nil {
		return fmt.Errorf("reading image: %w", err)
	}
	if err := os.MkdirAll(dest, perm); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	destP := path.Join(dest, n)
	if err := os.WriteFile(destP, b, perm); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}
	return nil
}

func lookupMainTemplate(content string) (*template.Template, error) {
	patterns := []string{
		path.Join(resources, "main.html"),
		path.Join(resources, "index.css"),
		path.Join(resources, "nav.html"),
		path.Join(resources, "nav.css"),
		content,
	}
	t := newTemplate("main.html")
	if _, err := t.ParseFS(siteFS, patterns...); err != nil {
		return nil, fmt.Errorf("parsing template filesystem: %w", err)
	}
	return t, nil
}

func newTemplate(tmplName string) *template.Template {
	t := template.New(tmplName)
	t.Option("missingkey=error")
	return t
}

type (
	Data struct {
		Site Site
		Page Page
	}
	Site struct {
		dest        string
		Name        string
		Description string
	}
	Page struct {
		Name string
		Data interface{}
	}
	PastEventYear struct {
		Year      string
		Events    bytes.Buffer
		Resources bytes.Buffer
	}
)

func (s *Site) addPage(destName, srcDir, srcName string, data interface{}) error {
	p := Page{
		Name: destName,
		Data: data,
	}
	tmplData := Data{
		Site: *s,
		Page: p,
	}
	if err := s.writeFile(srcDir, srcName, tmplData); err != nil {
		return fmt.Errorf("writing file %v, %w", srcName, err)
	}
	return nil
}

func (s *Site) addEvents() error {
	// TODO: calendar
	// TODO: future events
	// TODO href #anchor to year div s
	if err := s.addPastEvents(); err != nil {
		return fmt.Errorf("adding past events: %w", err)
	}
	return nil
}

func (s *Site) addPastEvents() error {
	eventsDir := path.Join(resources, "events", "past")
	yearEntries, err := siteFS.ReadDir(eventsDir)
	if err != nil {
		return fmt.Errorf("reading past events: %w", err)
	}
	slices.Reverse(yearEntries)
	var yrs []PastEventYear
	for _, y := range yearEntries {
		yr, err := s.addFolder(eventsDir, y)
		if err != nil {
			return fmt.Errorf("adding events for year %v: %w", y.Name(), err)
		}
		yrs = append(yrs, *yr)
	}
	if err := s.addPage("Past Events", "events", "past-events.html", yrs); err != nil {
		return fmt.Errorf("adding past events page: %w", err)
	}
	if err := s.addPage("Videos & Resources", "events", "videos-and-resources.html", yrs); err != nil {
		return fmt.Errorf("adding past events resources: %w", err)
	}
	return nil
}

func (s *Site) addFolder(dir string, f fs.DirEntry) (*PastEventYear, error) {
	folderName := f.Name()
	if !f.IsDir() {
		return nil, fmt.Errorf("unexpected folder: %v", folderName)
	}
	root := path.Join(dir, folderName)
	orderedFiles, err := siteFS.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading folder: %w", err)
	}
	slices.Reverse(orderedFiles)
	yr := PastEventYear{
		Year: folderName,
	}
	for _, ff := range orderedFiles {
		if err := yr.addFile(s, root, folderName, ff); err != nil {
			return nil, fmt.Errorf("adding file to event group: %w", err)
		}
	}
	return &yr, nil
}

func (yr *PastEventYear) addFile(s *Site, dir, year string, ff fs.DirEntry) error {
	nn := ff.Name()
	switch ext := path.Ext(nn); ext {
	case ".html":
		src := path.Join(dir, nn)
		if err := yr.addEvent(src); err != nil {
			return fmt.Errorf("adding event: %w", err)
		}
	case ".jpg", ".jpeg", ".png", ".webp":
		// TODO: resize images to max 125w, 150h
		dest := path.Join(s.dest, "images", "events", year)
		if err := addImage(ff, dir, dest); err != nil { // TODO: use less-generic version of addImage, providing data
			return fmt.Errorf("adding resource: %w", err)
		}
	case ".docx", ".pdf", ".ppt", ".pptx", ".xlsx":
		if err := yr.addResource(s, year, nn, dir); err != nil {
			return fmt.Errorf("adding resource: %w", err)
		}
	default:
		// this check is mostly for audit purposes
		// usually, add the extension to the list above
		return fmt.Errorf("unsupported file type: %q (%v)", ext, nn)
	}
	return nil
}

func (yr *PastEventYear) addEvent(src string) error {
	data, err := siteFS.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading event file: %w", err)
	}
	parts := []struct {
		tmplName string
		buf      *bytes.Buffer
	}{
		{"event", &yr.Events},
		{"resources", &yr.Resources},
	}
	for _, p := range parts {
		s := string(data)
		t := newTemplate(s)
		if _, err := t.Parse(s); err != nil {
			return fmt.Errorf("parsing event file: %w", err)
		}
		t = t.Lookup(p.tmplName)
		if t == nil {
			return fmt.Errorf("no template named %q in %v", p.tmplName, src)
		}
		if err := t.Execute(p.buf, nil); err != nil {
			return fmt.Errorf("executing template: %w", err)
		}
	}
	return nil
}

func (yr *PastEventYear) addResource(s *Site, year, name, dir string) error {
	srcP := path.Join(dir, name)
	b, err := siteFS.ReadFile(srcP)
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}
	dest := path.Join("resources", "events", year)
	destP := path.Join(s.dest, dest)
	if err := os.MkdirAll(destP, perm); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	destF := path.Join(destP, name)
	if err := os.WriteFile(destF, b, perm); err != nil {
		return fmt.Errorf("writing resource: %w", err)
	}
	return nil
}
