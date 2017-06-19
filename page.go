package gojekyll

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/osteele/gojekyll/helpers"
)

var (
	frontMatterMatcher     = regexp.MustCompile(`(?s)^---\n(.+?\n)---\n`)
	emptyFontMatterMatcher = regexp.MustCompile(`(?s)^---\n+---\n`)
)

// Page is a Jekyll page.
type Page interface {
	Path() string
	Site() *Site
	Source() string
	Static() bool
	Published() bool
	Permalink() string
	TemplateObject() VariableMap
	Write(io.Writer) error
	DebugVariables() VariableMap

	initPermalink() error
}

type pageFields struct {
	relpath     string // relative to site source, e.g. "_post/base.ext"
	permalink   string // cached permalink
	modTime     time.Time
	frontMatter VariableMap // page front matter, merged with defaults
	collection  *Collection
	site        *Site
}

func (p *pageFields) String() string {
	return fmt.Sprintf("%s{Path=%v, Permalink=%v}", reflect.TypeOf(p).Name(), p.relpath, p.permalink)
}

func (p *pageFields) Path() string      { return p.relpath }
func (p *pageFields) Permalink() string { return p.permalink }
func (p *pageFields) Published() bool   { return p.frontMatter.Bool("published", true) }
func (p *pageFields) Site() *Site       { return p.site }

// ReadPage reads a Page from a file, using defaults as the default front matter.
func ReadPage(site *Site, collection *Collection, relpath string, defaults VariableMap) (p Page, err error) {
	abspath := filepath.Join(site.Source, relpath)
	magic, err := helpers.ReadFileMagic(abspath)
	if err != nil {
		return
	}
	info, err := os.Stat(abspath)
	if err != nil {
		return
	}

	fields := pageFields{
		site:        site,
		collection:  collection,
		modTime:     info.ModTime(),
		relpath:     relpath,
		frontMatter: defaults,
	}
	if string(magic) == "---\n" {
		p, err = NewDynamicPage(fields)
		if err != nil {
			return
		}
	} else {
		p = &StaticPage{fields}
	}
	// Compute this after creating the page, in order to pick up the front matter.
	err = p.initPermalink()
	if err != nil {
		return
	}
	return
}

// TemplateObject returns the attributes of the template page object.
// See https://jekyllrb.com/docs/variables/#page-variables
func (p *pageFields) TemplateObject() VariableMap {
	var (
		relpath = "/" + filepath.ToSlash(p.relpath)
		base    = path.Base(relpath)
		ext     = path.Ext(relpath)
	)

	return VariableMap{
		"path":          relpath,
		"modified_time": p.modTime,
		"name":          base,
		"basename":      helpers.PathWithoutExtension(base),
		"extname":       ext,
	}
}

// DebugVariables returns a map that's useful to present during diagnostics.
// For a static page, this is just the page's template object attributes.
func (p *pageFields) DebugVariables() VariableMap {
	return p.TemplateObject()
}

// Source returns the file path of the page source.
func (p *pageFields) Source() string {
	return filepath.Join(p.site.Source, p.relpath)
}

// IsMarkdown returns a bool indicating whether the page is markdown.
func (p *pageFields) IsMarkdown() bool {
	return p.site.IsMarkdown(p.relpath)
}

// StaticPage is a static page.
type StaticPage struct {
	pageFields
}

// Static returns a bool indicating that the page is a static page.
func (page *StaticPage) Static() bool { return true }

// TemplateObject returns metadata for use in the representation of the page as a collection item
func (page *StaticPage) TemplateObject() VariableMap {
	return MergeVariableMaps(page.frontMatter, page.TemplateObject())
}

func (page *StaticPage) Write(w io.Writer) error {
	source, err := ioutil.ReadFile(page.Source())
	if err != nil {
		return err
	}
	_, err = w.Write(source)
	return err
}
