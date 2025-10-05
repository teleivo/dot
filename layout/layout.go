// Package layout TODO add godoc on all exported things
package layout

import (
	"fmt"
	"io"
	"strings"
)

// Format is the representation with which to render the layout.
type Format = int

const (
	// Default renders the layout content
	Default Format = iota
	// Layout renders the layout using an HTML like syntax
	Layout
	// Go renders the layout as GoString
	Go
)

var formats = map[string]Format{
	"default": Default,
	"layout":  Layout,
	"go":      Go,
}

func NewFormat(format string) (Format, error) {
	if f, ok := formats[format]; ok {
		return f, nil
	}
	return Default, fmt.Errorf("invalid format string %q", format)
}

type Doc struct {
	maxColumn int
	tags      []*tagInfo
}

func NewDoc(maxColumn int) *Doc {
	return &Doc{maxColumn: maxColumn}
}

type tagIterator func(yield func(*tagInfo, tagIterator) bool)

func (d *Doc) All() tagIterator {
	return d.newTagIterator(0, len(d.tags))
}

func (d *Doc) newTagIterator(i, j int) tagIterator {
	return func(yield func(*tagInfo, tagIterator) bool) {
		for i < j {
			if d.tags[i].len == 0 {
				if !yield(d.tags[i], d.newTagIterator(i, i)) {
					return
				}
				i++
			} else {
				if !yield(d.tags[i], d.newTagIterator(i+1, i+1+d.tags[i].len)) {
					return
				}
				i = i + 1 + d.tags[i].len
			}
		}
	}
}

func (d *Doc) Text(content string) *Doc {
	return d.tag(&text{content: content})
}

func (d *Doc) TextIf(content string, cond condition) *Doc {
	return d.tagIf(&text{content: content}, cond)
}

func (d *Doc) Space() *Doc {
	return d.tag(singleSpace)
}

func (d *Doc) SpaceIf(cond condition) *Doc {
	return d.tagIf(singleSpace, cond)
}

func (d *Doc) Break(count int) *Doc {
	return d.tag(newlines{count: count})
}

func (d *Doc) BreakIf(count int, cond condition) *Doc {
	return d.tagIf(newlines{count: count}, cond)
}

// Group a sequence of tags to be rendered as one line or multiple lines if they exceed the maximum
// column.
func (d *Doc) Group(body func(*Doc)) *Doc {
	return d.tagWith(&group{}, body)
}

// Indent a sequence of tags by given number of columns.
func (d *Doc) Indent(columns int, body func(*Doc)) *Doc {
	return d.IndentIf(columns, Always, body)
}

// IndentIf a sequence of tags by given number of columns if condition is met.
func (d *Doc) IndentIf(columns int, cond condition, body func(*Doc)) *Doc {
	return d.tagIfWith(&indentation{columns: columns}, cond, body)
}

func (d *Doc) tag(t tag) *Doc {
	return d.tagIfWith(t, Always, func(d *Doc) {})
}

func (d *Doc) tagIf(t tag, cond condition) *Doc {
	return d.tagIfWith(t, cond, func(d *Doc) {})
}

func (d *Doc) tagWith(t tag, body func(*Doc)) *Doc {
	return d.tagIfWith(t, Always, body)
}

func (d *Doc) tagIfWith(t tag, cond condition, body func(*Doc)) *Doc {
	i := len(d.tags)
	d.tags = append(d.tags, &tagInfo{tag: t, len: 0, cond: cond, measure: &measure{}})
	body(d)
	if j := len(d.tags); j != i {
		d.tags[i].len = j - i - 1
	}
	return d
}

func (d *Doc) Render(w io.Writer, format Format) error {
	d.measure()
	d.layout(d.All(), 0, 0)
	r := &renderer{w: w}

	var err error
	switch format {
	case Default:
		r.render(d.All(), true)
	case Layout:
		_, err = fmt.Fprint(w, d)
	case Go:
		goTemplate := `package main

import (
	"os"

	"github.com/teleivo/dot/layout"
)

func main() {
	d := %s
	d.Render(os.Stdout, layout.Default)
}
`
		_, err = fmt.Fprintf(w, goTemplate, goString(d, 1))
	}

	return err
}

type renderer struct {
	w        io.Writer // w writer to output formatted DOT code to
	indent   int       // indent is the current level of indentation
	space    bool      // space indicates a buffered space that should be rendered
	newlines int       // newlines indicates the number of buffered newline that should be rendered
}

func (d *Doc) measure() {
	for t, children := range d.All() {
		measureIter(t, children)
	}
	for t, children := range d.All() {
		sumWidths(t, children)
	}
}

func measureIter(parent *tagInfo, children tagIterator) {
	tagWidth(parent)
	for t, children := range children {
		measureIter(t, children)
	}
}

func tagWidth(t *tagInfo) {
	if t.cond == Broken { // only measure flat width
		return
	}

	switch tag := t.tag.(type) {
	case *text:
		t.measure.width = len(tag.content)
	case space:
		t.measure.width = 1
	case newlines:
		t.measure.broken = true
	}
}

func sumWidths(parent *tagInfo, children tagIterator) measure {
	for t, children := range children {
		parent.measure.add(sumWidths(t, children))
	}
	return *parent.measure
}

func (d *Doc) layout(iter tagIterator, indent, column int) {
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *group:
			if t.measure.width > d.maxColumn {
				t.measure.broken = true
			}
			d.layout(children, indent, column)
		case *indentation:
			if t.cond != Flat {
				// TODO implement safety on under/overflow
				d.layout(children, indent+tag.columns, column)
			}
		case *text:
			column += len(tag.content)
		case space:
			column++
		case newlines:
			// TODO reset width except 0 newlines? what does Break(0) mean?
			column = indent
		}
	}
}

func (r *renderer) render(iter tagIterator, isParentBroken bool) {
	for t, children := range iter {
		if t.cond == Flat && isParentBroken || t.cond == Broken && !isParentBroken {
			continue
		}

		switch tag := t.tag.(type) {
		case *group:
			r.render(children, t.measure.broken)
		case *indentation:
			// TODO implement indentation, only indent if we have pending newline(s)?
			// TODO implement safety on under/overflow
			r.indent += tag.columns
			r.render(children, isParentBroken)
			r.indent -= tag.columns
		case *text:
			if r.newlines == 0 && r.space { // prevents trailing whitespace
				fmt.Fprintf(r.w, " ")
				r.space = false
			}
			// TODO is batching prints more efficient? like having a slice of 10 newlines and
			// printing at least up to 10 at a time?
			for i := r.newlines; i > 0; i-- {
				fmt.Fprintf(r.w, "\n")
			}
			if r.newlines > 0 {
				for i := r.indent; i > 0; i-- {
					fmt.Fprintf(r.w, "\t")
				}
			}
			r.newlines = 0
			fmt.Fprintf(r.w, "%s", tag.content)
		case space:
			r.space = true
		case newlines:
			r.newlines += tag.count
		}
	}
}

func (d *Doc) String() string {
	var sb strings.Builder
	stringIter(&sb, d.All(), 0)
	return sb.String()
}

func stringIter(w io.Writer, iter tagIterator, indent int) {
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *group:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<group width=%s>\n", t.measure)
			stringIter(w, children, indent+1)
			writeIndent(w, indent)
			fmt.Fprintf(w, "</group>\n")
		case *indentation:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<indent columns=%d>\n", tag.columns)
			stringIter(w, children, indent+1)
			writeIndent(w, indent)
			fmt.Fprintf(w, "</indent>\n")
		case *text:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<text width=%s content=%q/>\n", t.measure, tag.content)
		case space:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<space/>\n")
		case newlines:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<break count=%d/>\n", tag.count)
		}
	}
}

func writeIndent(w io.Writer, columns int) {
	for range columns {
		fmt.Fprint(w, "\t")
	}
}

func (d *Doc) GoString() string {
	return goString(d, 0)
}

func goString(d *Doc, indent int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "layout.NewDoc(%d)\n", d.maxColumn)
	goStringIter(&sb, d.All(), indent)
	return sb.String()
}

// TODO can I reduce fmt calls?
// TODO make simple test for GoStringer/String on literal want structures
func goStringIter(w io.Writer, iter tagIterator, indent int) {
	first := true
	for t, children := range iter {
		if first {
			writeIndent(w, indent)
			fmt.Fprint(w, "d.\n")
			indent++
		} else {
			fmt.Fprint(w, ".\n")
		}
		writeIndent(w, indent)

		switch tag := t.tag.(type) {
		case *group:
			fmt.Fprint(w, "Group(func(d *layout.Doc) {\n")
			goStringIter(w, children, indent+1)
			fmt.Fprintln(w)
			writeIndent(w, indent)
			fmt.Fprintf(w, "})")
		case *indentation:
			if t.cond == Always {
				fmt.Fprintf(w, "Indent(%d, func(d *layout.Doc) {\n", tag.columns)
			} else {
				fmt.Fprintf(w, "IndentIf(%d, layout.%s, func(d *layout.Doc) {\n", tag.columns, t.cond)
			}
			goStringIter(w, children, indent+1)
			fmt.Fprintln(w)
			writeIndent(w, indent)
			fmt.Fprint(w, "})")
		case *text:
			if t.cond == Always {
				fmt.Fprintf(w, "Text(%q)", tag.content)
			} else {
				fmt.Fprintf(w, "TextIf(%q, layout.%s)", tag.content, t.cond)
			}
		case space:
			if t.cond == Always {
				fmt.Fprint(w, "Space()")
			} else {
				fmt.Fprintf(w, "SpaceIf(layout.%s)", t.cond)
			}
		case newlines:
			if t.cond == Always {
				fmt.Fprintf(w, "Break(%d)", tag.count)
			} else {
				fmt.Fprintf(w, "BreakIf(%d, layout.%s)", tag.count, t.cond)
			}
		}
		first = false
	}
}

type condition int

const (
	Always condition = iota
	Flat
	Broken
)

func (c condition) String() string {
	switch c {
	case Always:
		return "Always"
	case Flat:
		return "Flat"
	case Broken:
		return "Broken"
	default:
		panic("condition string not implemented")
	}
}

// TODO what is the benefit of wrapping Tag? is it so a Tag is the API and users cannot mess with
// measurement and len? can I achieve that without yet another type

type tagInfo struct {
	tag     tag
	len     int
	cond    condition
	measure *measure
}

func (t *tagInfo) String() string {
	return fmt.Sprintf("TagInfo{tag=%s, len=%d, cond=%s, measure=%s}", t.tag, t.len, t.cond, t.measure)
}

type measure struct {
	width  int
	broken bool
}

func (m *measure) add(b measure) {
	if m.broken || b.broken {
		m.broken = true
	} else {
		m.width += b.width
	}
}

func (m *measure) String() string {
	if m.broken {
		return "broken"
	}
	return fmt.Sprint(m.width)
}

type tag interface {
	tag()
}

// Group a sequence of tags to be rendered as one line or multiple lines if they exceed the maximum
// column.
type group struct{}

func (g *group) tag() {}

func (g *group) String() string {
	return "Group"
}

type indentation struct {
	columns int
}

func (i *indentation) tag() {}

func (i *indentation) String() string {
	return fmt.Sprintf("Indent(%d)", i.columns)
}

type text struct {
	content string
}

func (t *text) tag() {}

func (t *text) String() string {
	return fmt.Sprintf("Text(%q)", t.content)
}

var singleSpace = space{}

type space struct{}

func (s space) tag() {}

func (s space) String() string {
	return "Space"
}

type newlines struct {
	count int
}

func (n newlines) tag() {}

func (n newlines) String() string {
	return fmt.Sprintf("Break(%d)", n.count)
}
