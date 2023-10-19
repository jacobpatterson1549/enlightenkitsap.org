package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"text/template"
)

type (
	Data struct {
		Site Site
		Page Page
	}
	Site struct {
		fSys        fs.FS
		dest        string
		Name        string
		Description string
		removeAll   func(path string) error
		mkdirAll    func(path string) error
		writeFile   func(name string, data []byte) error
		isNotExist  func(err error) bool
	}
	Page struct {
		Name string
		Data interface{}
	}
	EventGroup struct {
		Year      string
		Events    bytes.Buffer
		Resources bytes.Buffer
	}
)

func (s *Site) addMain() error {
	pages := []struct {
		srcDir   string
		fileName string
		name     string
	}{
		{"", "home", "Home Page"},
		{"about", "board-members", "Board Members"},
		{"about", "contact-us", "Contact Us"},
		{"about", "donations", "Donations"},
		{"about", "location", "Where Are We Located?"},
		{"about", "mission-statement", "Mission Statement"},
		{"about", "purpose-statement", "Purpose Statement"},
		{"about", "volunteers", "Volunteers"},
		{"events", "calendar", "Calendar"},
		{"events", "sign-up", "Sign Up For Events"},
	}
	for _, pg := range pages {
		if err := s.addPage(pg.name, pg.srcDir, pg.fileName+".html", nil); err != nil {
			return fmt.Errorf("writing page: %w", err)
		}
	}
	imageDirs := []struct {
		src     string
		dest    string
		maxSize int
	}{
		{"", "", kB50}, // root images from resources
		{"about", "board", kB50},
	}
	for _, img := range imageDirs {
		src := path.Join(resources, img.src, "images")
		destDir := path.Join("images", img.dest)
		if err := s.addImages(src, destDir, img.maxSize); err != nil {
			return fmt.Errorf("adding images from: %w", err)
		}
	}
	return nil
}

func (s *Site) cleanDest() error {
	if err := s.removeAll(s.dest); err != nil && !s.isNotExist(err) {
		return fmt.Errorf("removing old version of site: %w", err)
	}
	if err := s.mkdirAll(s.dest); err != nil {
		return fmt.Errorf("creating new site directory: %w", err)
	}
	return nil
}

func (s *Site) addImages(srcDir, destDir string, maxSize int) error {
	entries, err := fs.ReadDir(s.fSys, srcDir)
	if err != nil {
		return fmt.Errorf("reading image directory: %w", err)
	}
	if err := s.mkdirAll(destDir); err != nil {
		return fmt.Errorf("creating image directory: %w", err)
	}
	for _, f := range entries {
		nn := f.Name()
		if f.IsDir() {
			return fmt.Errorf("unexpected directory for images: %q", nn)
		}
		switch ext := path.Ext(nn); ext {
		case ".png", ".jpg":
			if err := s.addImage(f, srcDir, destDir, maxSize); err != nil {
				return fmt.Errorf("adding image: %w", err)
			}
		default:
			return fmt.Errorf("unexpected image extension: %q (%q)", ext, nn)
		}
	}
	return nil
}

func (s *Site) addImage(f fs.DirEntry, src, destDir string, maxSize int) error {
	if f.IsDir() {
		return fmt.Errorf("will not read directory from image folder")
	}
	n := f.Name()
	srcP := path.Join(src, n)
	b, err := fs.ReadFile(s.fSys, srcP)
	if len(b) > maxSize && maxSize > 0 {
		return fmt.Errorf("image %q larger than %v bytes", n, maxSize)
	}
	if err != nil {
		return fmt.Errorf("reading image: %w", err)
	}
	dest := path.Join(s.dest, destDir)
	if err := s.mkdirAll(dest); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	destP := path.Join(dest, n)
	if err := s.writeFile(destP, b); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}
	return nil
}

func (s *Site) addPage(pageName, srcDir, srcName string, data interface{}) error {
	p := Page{
		Name: pageName,
		Data: data,
	}
	tmplData := Data{
		Site: *s,
		Page: p,
	}
	if err := s.addFile(srcDir, srcName, tmplData); err != nil {
		return fmt.Errorf("writing file %v, %w", srcName, err)
	}
	return nil
}

func (s *Site) addFile(srcDir, name string, data interface{}) error {
	if err := s.mkdirAll(s.dest); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	src := path.Join(resources, srcDir, name)
	t, err := s.lookupMainTemplate(src)
	if err != nil {
		return fmt.Errorf("looking up template: %w", err)
	}
	buf := new(bytes.Buffer)
	if err := s.executeTemplate(buf, t, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	b := buf.Bytes()
	dest := path.Join(s.dest, name)
	if err := s.writeFile(dest, b); err != nil {
		return fmt.Errorf("writing template: %w", err)
	}
	return nil
}

func (s *Site) lookupMainTemplate(content string) (*template.Template, error) {
	patterns := []string{
		path.Join(resources, "main.html"),
		path.Join(resources, "index.css"),
		path.Join(resources, "nav.html"),
		path.Join(resources, "nav.css"),
		content,
	}
	t := s.newTemplate("main.html")
	if _, err := t.ParseFS(s.fSys, patterns...); err != nil {
		return nil, fmt.Errorf("parsing template filesystem: %w", err)
	}
	return t, nil
}

func (*Site) newTemplate(tmplName string) *template.Template {
	t := template.New(tmplName)
	t.Option("missingkey=error")
	return t
}

func (*Site) executeTemplate(w io.Writer, t *template.Template, data interface{}) error {
	sb := new(strings.Builder)
	if err := t.Execute(sb, data); err != nil {
		return fmt.Errorf("executing template to buffer: %w", err)
	}
	got := sb.String()
	thin := strings.TrimSpace(got)
	r := strings.NewReader(thin)
	if _, err := r.WriteTo(w); err != nil {
		return fmt.Errorf("executing template buffer to target: %w", err)
	}
	return nil
}

func (s *Site) addEvents() error {
	if err := s.addFutureEvents(); err != nil {
		return fmt.Errorf("adding future events: %w", err)
	}
	if err := s.addPastEvents(); err != nil {
		return fmt.Errorf("adding past events: %w", err)
	}
	return nil
}

func (s *Site) addFutureEvents() error {
	eventsDir := path.Join(resources, "events")
	eventEntries, err := fs.ReadDir(s.fSys, eventsDir)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	idx := slices.IndexFunc(eventEntries, func(de fs.DirEntry) bool {
		n := de.Name()
		return n == "future"
	})
	if idx < 0 {
		return fmt.Errorf("futureEvents directory not found")
	}
	futureEntry := eventEntries[idx]
	e, err := s.createEventGroup(eventsDir, futureEntry)
	if err != nil {
		return fmt.Errorf("adding future events folder: %w", err)
	}
	if err := s.addPage("Upcoming Speakers", "events", "future-events.html", e); err != nil {
		return fmt.Errorf("adding future events page: %w", err)
	}
	return err
}

func (s *Site) addPastEvents() error {
	eventsDir := path.Join(resources, "events", "past")
	yearEntries, err := fs.ReadDir(s.fSys, eventsDir)
	if err != nil {
		return fmt.Errorf("reading past events: %w", err)
	}
	slices.Reverse(yearEntries)
	yrs := make([]EventGroup, 0, len(yearEntries))
	for _, y := range yearEntries {
		yr, err := s.createEventGroup(eventsDir, y)
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

func (s *Site) createEventGroup(dir string, f fs.DirEntry) (*EventGroup, error) {
	folderName := f.Name()
	if !f.IsDir() {
		return nil, fmt.Errorf("unexpected folder: %v", folderName)
	}
	root := path.Join(dir, folderName)
	orderedFiles, err := fs.ReadDir(s.fSys, root)
	if err != nil {
		return nil, fmt.Errorf("reading folder: %w", err)
	}
	slices.Reverse(orderedFiles)
	eg := new(EventGroup)
	eg.Year = folderName
	for _, ff := range orderedFiles {
		if err := s.addEventFile(eg, root, folderName, ff); err != nil {
			return nil, fmt.Errorf("adding file to event group: %w", err)
		}
	}
	return eg, nil
}

func (s *Site) addEventFile(eg *EventGroup, dir, year string, ff fs.DirEntry) error {
	nn := ff.Name()
	switch ext := path.Ext(nn); ext {
	case ".html":
		src := path.Join(dir, nn)
		if err := s.addEvent(eg, src); err != nil {
			return fmt.Errorf("adding event: %w", err)
		}
	case ".jpg":
		destDir := path.Join("images", "events", year)
		if err := s.addImage(ff, dir, destDir, kB50); err != nil {
			return fmt.Errorf("adding resource: %w", err)
		}
	case ".docx", ".pdf", ".ppt", ".pptx", ".xlsx":
		if err := s.addResource(year, nn, dir); err != nil {
			return fmt.Errorf("adding resource: %w", err)
		}
	default:
		// this check is mostly for audit purposes
		// usually, add the extension to the list above
		return fmt.Errorf("unsupported file type: %q (%v)", ext, nn)
	}
	return nil
}

func (s *Site) addEvent(eg *EventGroup, src string) error {
	data, err := fs.ReadFile(s.fSys, src)
	if err != nil {
		return fmt.Errorf("reading event file: %w", err)
	}
	parts := []struct {
		tmplName string
		buf      *bytes.Buffer
	}{
		{"event", &eg.Events},
		{"resources", &eg.Resources},
	}
	for _, p := range parts {
		t := s.newTemplate("")
		if _, err := t.Parse(string(data)); err != nil {
			return fmt.Errorf("parsing event file: %w", err)
		}
		t = t.Lookup(p.tmplName)
		if t == nil {
			return fmt.Errorf("no template named %q in %v", p.tmplName, src)
		}
		if err := s.executeTemplate(p.buf, t, nil); err != nil {
			return fmt.Errorf("executing template: %w", err)
		}
	}
	return nil
}

func (s *Site) addResource(year, name, dir string) error {
	srcP := path.Join(dir, name)
	b, err := fs.ReadFile(s.fSys, srcP)
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}
	dest := path.Join("resources", "events", year)
	destP := path.Join(s.dest, dest)
	if err := s.mkdirAll(destP); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	destF := path.Join(destP, name)
	if err := s.writeFile(destF, b); err != nil {
		return fmt.Errorf("writing resource: %w", err)
	}
	return nil
}
