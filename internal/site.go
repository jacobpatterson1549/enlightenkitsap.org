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
		OneResource bool
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
		{about, "board-members", "Board Members"},
		{about, "contact-us", "Contact Us"},
		{about, "donations", "Donations"},
		{about, "location", "Where Are We Located?"},
		{about, "mission-statement", "Mission Statement"},
		{about, "purpose-statement", "Purpose Statement"},
		{about, "volunteers", "Volunteers"},
		{events, "calendar", "Calendar"},
		{events, "sign-up", "Sign Up For Events"},
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
		{about, "board", kB50},
	}
	for _, img := range imageDirs {
		src := path.Join(resources, img.src, "images")
		destDir := path.Join("images", img.dest)
		if err := s.addImages(src, destDir, img.maxSize); err != nil {
			return fmt.Errorf("adding images from: %w", err)
		}
	}
	if err := s.addStatic("", "", "robots.txt"); err != nil {
		return fmt.Errorf("adding robots.txt: %w", err)
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

func (s *Site) addStatic(srcDir, destDir, name string) error {
	src := path.Join(resources, srcDir, name)
	dest := path.Join(s.dest, destDir, name)
	data, err := fs.ReadFile(s.fSys, src)
	if err != nil {
		return fmt.Errorf("opening static file: %w", err)
	}
	if err := s.writeFile(dest, data); err != nil {
		return fmt.Errorf("writing static file: %w", err)
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
	eventsDir := path.Join(resources, events)
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
	if err := s.addPage("Upcoming Speakers", events, "future-events.html", e); err != nil {
		return fmt.Errorf("adding future events page: %w", err)
	}
	return err
}

func (s *Site) addPastEvents() error {
	eventsDir := path.Join(resources, events, "past")
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
	if err := s.addPage("Past Events", events, "past-events.html", yrs); err != nil {
		return fmt.Errorf("adding past events page: %w", err)
	}
	if s.OneResource {
		if err := s.addPage("Videos & Resources", events, "videos-and-resources.html", yrs); err != nil {
			return fmt.Errorf("adding past events resources: %w", err)
		}
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
		if err := s.addEvent(eg, dir, nn, year); err != nil {
			return fmt.Errorf("adding event: %w", err)
		}
	case ".jpg":
		destDir := path.Join("images", events, year)
		if err := s.addImage(ff, dir, destDir, kB50); err != nil {
			return fmt.Errorf("adding resource: %w", err)
		}
	case ".pdf", ".docx", ".xlsx":
		destDir := path.Join("resources", "events", year)
		if err := s.addImage(ff, dir, destDir, mB10); err != nil {
			return fmt.Errorf("adding resource: %w", err)
		}
	default:
		// this check is mostly for audit purposes
		// usually, add the extension to the list above
		return fmt.Errorf("unsupported file type: %q (%v)", ext, nn)
	}
	return nil
}

func (s *Site) addEvent(eg *EventGroup, dir, eventHtmlName, year string) error {
	src := path.Join(dir, eventHtmlName)
	data, err := fs.ReadFile(s.fSys, src)
	if err != nil {
		return fmt.Errorf("reading event file: %w", err)
	}
	parts := []struct {
		tmplName string
		buf      *bytes.Buffer
	}{
		{"event", &eg.Events},
		{"resources", func() *bytes.Buffer {
			if !s.OneResource {
				return new(bytes.Buffer)
			}
			return &eg.Resources
		}(),
		},
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
		beforeLen := p.buf.Len()
		if err := s.executeTemplate(p.buf, t, nil); err != nil {
			return fmt.Errorf("executing template: %w", err)
		}
		afterLen := p.buf.Len()
		if p.tmplName == "resources" && beforeLen != afterLen && !s.OneResource {
			if err := s.addResourcesLink(year, eventHtmlName, &eg.Events, p.buf); err != nil {
				return fmt.Errorf("adding resources link: %w", err)
			}
		}
	}
	return nil
}

func (s *Site) addResourcesLink(year, eventHtmlName string, eventBuf, resourcesBuf *bytes.Buffer) error {
	dest := path.Join(resources, events, year)
	destP := path.Join(s.dest, dest)
	resourceName := path.Join(destP, eventHtmlName)
	linkHref := path.Join(dest, eventHtmlName)
	if err := s.addEventResourcesPage(destP, resourceName, resourcesBuf); err != nil {
		return fmt.Errorf("adding event resources page: %w", err)
	}
	if err := s.addEventResourcesLink(linkHref, eventBuf); err != nil {
		return fmt.Errorf("adding event resources link: %w", err)
	}
	return nil
}

func (s *Site) addEventResourcesPage(destP, resourceName string, resourcesBuf *bytes.Buffer) error {
	if err := s.mkdirAll(destP); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}
	// t, err := s.lookupMainTemplate("")
	// if err != nil {
	// 	return fmt.Errorf("looking event resources template: %w", err)
	// }
	// TODO: this next chunk is a copy from lookupMainTemplate().  fix the duplication
	patterns := []string{
		path.Join(resources, "main.html"),
		path.Join(resources, "index.css"),
		path.Join(resources, "nav.html"),
		path.Join(resources, "nav.css"),
	}
	t := s.newTemplate("main.html")
	if _, err := t.ParseFS(s.fSys, patterns...); err != nil {
		return fmt.Errorf("parsing main template filesystem: %w", err)
	}

	content := new(bytes.Buffer)
	content.WriteString(`{{define "content"}}`)
	resourcesBuf.WriteTo(content)
	content.WriteString(`<div class="left">`)
	content.WriteString(`<a href="javascript:history.back()">back</a>`)
	content.WriteString(`</div>`)
	content.WriteString(`{{end}}`)
	contentTmpl := content.String()

	if _, err := t.Parse(contentTmpl); err != nil {
		return fmt.Errorf("parsing content template: %w", err)
	}
	buf2 := new(bytes.Buffer)
	// TODO: this is similar to Site.addPage()
	p := Page{
		Name: "Videos/Resources for Event",
	}
	tmplData := Data{
		Site: *s,
		Page: p,
	}
	if err := s.executeTemplate(buf2, t, tmplData); err != nil {
		return fmt.Errorf("writing resources info template: %w", err)
	}
	data := buf2.Bytes()
	if err := s.writeFile(resourceName, data); err != nil {
		return fmt.Errorf("writing resources file for event: %w", err)
	}
	return nil
}

func (s *Site) addEventResourcesLink(linkHref string, eventBuf *bytes.Buffer) error {
	// TODO: cache the link template
	eventLinkPath := path.Join(resources, events, "past-events.html")
	t := s.newTemplate("")
	if _, err := t.ParseFS(s.fSys, eventLinkPath); err != nil {
		return fmt.Errorf("parsing event resources link template: %w", err)
	}
	t = t.Lookup("event-resource-link")
	if t == nil {
		return fmt.Errorf("event-resource-link template not found")
	}
	if err := s.executeTemplate(eventBuf, t, linkHref); err != nil {
		return fmt.Errorf("writing event link template: %w", err)
	}
	return nil
}
