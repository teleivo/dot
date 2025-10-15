// Package layout provides a declarative toolkit for building pretty printers and code formatters.
//
// It implements a DOM-like structure that specifies how text should be laid out with respect to
// line breaking, indentation, and reflowing. The core abstraction is [Doc], a tree of tags that
// describe layout constraints rather than explicit formatting decisions.
//
// A [Doc] is built by chaining method calls that add tags:
//   - [Doc.Text]: adds literal text content
//   - [Doc.Space]: adds a single space
//   - [Doc.Break]: adds one or more newlines
//   - [Doc.Group]: marks a sequence of tags that should be kept on one line if possible
//   - [Doc.Indent]: increases indentation level for a sequence of tags
//
// Tags can be conditional using the *If variants ([Doc.TextIf], [Doc.SpaceIf], [Doc.BreakIf]),
// which only render when a containing group is either flat (fits on one line) or broken (spans
// multiple lines).
//
// The layout engine uses a two-phase approach:
//
//  1. Measure: computes the width of each group assuming no internal line breaks
//  2. Layout: determines which groups must break based on the maximum column width
//
// Groups are broken if either their measured width exceeds the remaining space on the current
// line, or they contain inherent newlines (from [Doc.Break]). Breaking decisions propagate
// outward: a broken inner group forces its parent to break as well.
//
// # Acknowledgments
//
// This package is a Go port of [allman] by mcyoung. The layout algorithm and design are based on
// the excellent article ["The Art of Formatting Code"].
//
// [allman]: https://github.com/mcy/strings/tree/main/allman
// ["The Art of Formatting Code"]: https://mcyoung.xyz/2025/03/11/formatters/
package layout

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// Format specifies the output representation for rendering a [Doc].
type Format = int

const (
	// Default renders the formatted output as text.
	Default Format = iota
	// Layout renders the document structure using HTML-like syntax, showing all tags including
	// those that may not appear in the final output. This is useful for debugging the measure
	// and layout algorithm to understand why a group breaks.
	Layout
	// Go renders the document as a runnable Go program that reproduces the layout as rendered
	// by [Default]. This enables debugging and iteration on layouts.
	Go
)

var formats = map[string]Format{
	"default": Default,
	"go":      Go,
	"layout":  Layout,
}

var validFormats = [3]string{"default", "go", "layout"}

// NewFormat converts a string to a [Format] constant. Valid values are "default", "layout", and
// "go". Returns an error if the format string is invalid.
func NewFormat(format string) (Format, error) {
	if f, ok := formats[format]; ok {
		return f, nil
	}
	return Default, fmt.Errorf("invalid format string: %q, valid ones are: %q", format, validFormats)
}

// Doc represents a document for layout formatting. Build it by chaining method calls like
// [Doc.Text], [Doc.Space], [Doc.Break], [Doc.Group], and [Doc.Indent]. Render it using
// [Doc.Render]. Note that rendering mutates the document, so use [Doc.Clone] to create a copy
// if you need to render multiple times.
type Doc struct {
	maxColumn int
	tags      []*node
}

// NewDoc creates a new document with the specified maximum column width. Text will be reflowed
// to fit within this width where possible.
func NewDoc(maxColumn int) *Doc {
	return &Doc{maxColumn: maxColumn}
}

// Clone creates a deep copy of the Doc. Use this if you want to [Doc.Render] a Doc multiple times.
func (d *Doc) Clone() *Doc {
	clone := &Doc{
		maxColumn: d.maxColumn,
		tags:      make([]*node, len(d.tags)),
	}
	for i, t := range d.tags {
		clone.tags[i] = &node{
			tag:     t.tag,
			len:     t.len,
			cond:    t.cond,
			measure: &measure{},
		}
	}
	return clone
}

type tagIterator func(yield func(*node, tagIterator) bool)

// All returns an iterator over all tags in the document. This is used internally by the layout
// engine and for implementing [Doc.String] and [Doc.GoString].
func (d *Doc) All() tagIterator {
	return d.newTagIterator(0, len(d.tags))
}

func (d *Doc) newTagIterator(i, j int) tagIterator {
	return func(yield func(*node, tagIterator) bool) {
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

// Text adds literal text content to the document.
func (d *Doc) Text(content string) *Doc {
	return d.tag(&text{content: content})
}

// TextIf adds literal text content that only renders when the specified condition is met.
func (d *Doc) TextIf(content string, cond condition) *Doc {
	return d.tagIf(&text{content: content}, cond)
}

// Space adds a single space to the document.
func (d *Doc) Space() *Doc {
	return d.tag(singleSpace)
}

// SpaceIf adds a single space that only renders when the specified condition is met.
func (d *Doc) SpaceIf(cond condition) *Doc {
	return d.tagIf(singleSpace, cond)
}

// Break adds one or more newlines to the document. The count must be positive.
func (d *Doc) Break(count int) *Doc {
	if count <= 0 {
		panic("Break: count must be positive")
	}
	return d.tag(newlines{count: count})
}

// BreakIf adds one or more newlines that only render when the specified condition is met.
// The count must be positive.
func (d *Doc) BreakIf(count int, cond condition) *Doc {
	if count <= 0 {
		panic("BreakIf: count must be positive")
	}
	return d.tagIf(newlines{count: count}, cond)
}

// Group marks a sequence of content that should be kept on one line if it fits within the
// maximum column width, or broken across multiple lines if it doesn't.
func (d *Doc) Group(body func(*Doc)) *Doc {
	return d.tagWith(&group{}, body)
}

// Indent increases the indentation level by the specified number of columns for the content
// added in body. The indentation is applied at the start of each line after a newline.
// Each column of indentation is rendered as a single tab character.
func (d *Doc) Indent(columns int, body func(*Doc)) *Doc {
	return d.tagWith(&indentation{columns: columns}, body)
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

	// merge consecutive spaces of the same condition
	if _, ok := t.(space); ok && i > 0 {
		if _, ok := d.tags[i-1].tag.(space); ok && cond == d.tags[i-1].cond {
			return d
		}
	}

	d.tags = append(d.tags, &node{tag: t, len: 0, cond: cond, measure: &measure{}})
	body(d)
	if j := len(d.tags); j != i {
		d.tags[i].len = j - i - 1
	}
	return d
}

// Render writes the formatted document to the writer in the specified format. Note that rendering
// mutates the document, so re-rendering the same document will produce incorrect results. Use
// [Doc.Clone] to create a copy if you need to render multiple times or to different outputs.
func (d *Doc) Render(w io.Writer, format Format) error {
	d.measure()
	d.layout(d.All(), 0, 0)
	r := &renderer{w: w}

	var err error
	switch format {
	case Default:
		err = r.render(d.All(), true)
	case Layout:
		_, err = fmt.Fprint(w, d)
	case Go:
		goTemplate := `package main

import (
	"os"

	"github.com/teleivo/dot/internal/layout"
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
	w               io.Writer // w writer to output formatted DOT code to
	indent          int       // indent is the current level of indentation
	pendingSpace    bool      // pendingSpace indicates a space that will only be rendered if its not trailing
	writtenNewlines int       // writtenNewlines indicates the number of newlines that were written to merge consecutive newlines
}

func (r *renderer) write(s string) error {
	_, err := io.WriteString(r.w, s)
	return err
}

func (d *Doc) measure() {
	for t, children := range d.All() {
		measureIter(t, children)
	}
	for t, children := range d.All() {
		sumWidths(t, children)
	}
}

func measureIter(parent *node, children tagIterator) {
	tagWidth(parent)
	for t, children := range children {
		measureIter(t, children)
	}
}

func tagWidth(t *node) {
	if t.cond == Broken { // only measure flat width
		return
	}

	switch tag := t.tag.(type) {
	case *text:
		t.measure.width = len(tag.content)
	case space:
		// Spaces start as pending - they'll be included in width during sumWidths if
		// followed by content
		t.measure.pendingSpace = true
	case newlines:
		t.measure.broken = true
	}
}

func sumWidths(parent *node, children tagIterator) measure {
	for t, children := range children {
		child := sumWidths(t, children)
		parent.measure.add(child)
	}
	return *parent.measure
}

func (d *Doc) layout(iter tagIterator, indent, column int) {
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *group:
			if t.measure.broken || column+t.measure.width > d.maxColumn {
				t.measure.broken = true
				d.layout(children, indent, column)
			} else {
				column += t.measure.width
			}
		case *indentation:
			d.layout(children, safeAdd(indent, tag.columns), column)
		case *text:
			column += len(tag.content)
		case space:
			column++
		case newlines:
			column = indent
		}
	}
}

func safeAdd(a, b int) int {
	if b > 0 && a > math.MaxInt-b {
		panic(fmt.Errorf("overflow adding %d to %d", a, b))
	}
	if b < 0 && a < math.MinInt-b {
		panic(fmt.Errorf("underflow adding %d to %d", a, b))
	}

	return a + b
}

func (r *renderer) render(iter tagIterator, isParentBroken bool) error {
	for t, children := range iter {
		if t.cond == Flat && isParentBroken || t.cond == Broken && !isParentBroken {
			continue
		}

		switch tag := t.tag.(type) {
		case *group:
			if err := r.render(children, t.measure.broken); err != nil {
				return err
			}
		case *indentation:
			r.indent = safeAdd(r.indent, tag.columns)
			if err := r.render(children, isParentBroken); err != nil {
				return err
			}
			r.indent -= tag.columns
		case *text:
			if r.pendingSpace { // space is not trailing so write it
				if err := r.write(" "); err != nil {
					return err
				}
				r.pendingSpace = false
			}
			if r.writtenNewlines > 0 {
				for i := r.indent; i > 0; i-- {
					if err := r.write("\t"); err != nil {
						return err
					}
				}
			}
			if err := r.write(tag.content); err != nil {
				return err
			}
			r.writtenNewlines = 0 // reset newlines as text means we do not deal with consecutive newlines
		case space:
			r.pendingSpace = true // writing space is delayed as it might be trailing
		case newlines:
			r.pendingSpace = false // discard pending space which would be trailing
			// merge consecutive Breaks
			for ; r.writtenNewlines < tag.count; r.writtenNewlines++ {
				if err := r.write("\n"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// String returns the document structure as HTML-like markup, showing all tags and their properties.
// This implements [fmt.Stringer] and is like rendering with [Layout] format except that the measure
// and layout phases are not run. Useful for debugging the layout algorithm.
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
			switch t.cond { // width is not computed for text that only renders when layout is Broken
			case Always:
				fmt.Fprintf(w, "<text width=%s content=%q/>\n", t.measure, tag.content)
			case Flat:
				fmt.Fprintf(w, "<text cond=%q width=%s content=%q/>\n", t.cond, t.measure, tag.content)
			default:
				fmt.Fprintf(w, "<text cond=%q content=%q/>\n", t.cond, tag.content)
			}
		case space:
			writeIndent(w, indent)
			if t.cond == Always {
				fmt.Fprintf(w, "<space/>\n")
			} else {
				fmt.Fprintf(w, "<space cond=%q/>\n", t.cond)
			}
		case newlines:
			writeIndent(w, indent)
			if t.cond == Always {
				fmt.Fprintf(w, "<break count=%d/>\n", tag.count)
			} else {
				fmt.Fprintf(w, "<break cond=%q count=%d/>\n", t.cond, tag.count)
			}
		}
	}
}

func writeIndent(w io.Writer, columns int) {
	for range columns {
		fmt.Fprint(w, "\t")
	}
}

// GoString returns the document as runnable Go code that reproduces the layout. This implements
// [fmt.GoStringer] and is like rendering with [Go] format except that the measure and layout phase
// are not run. Useful for debugging and iterating on layouts by generating standalone programs.
func (d *Doc) GoString() string {
	return goString(d, 0)
}

func goString(d *Doc, indent int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "layout.NewDoc(%d)\n", d.maxColumn)
	goStringIter(&sb, d.All(), indent)
	return sb.String()
}

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
			fmt.Fprintf(w, "Indent(%d, func(d *layout.Doc) {\n", tag.columns)
			goStringIter(w, children, indent+1)
			fmt.Fprintln(w)
			writeIndent(w, indent)
			fmt.Fprint(w, "})")
		case *text:
			if t.cond == Always {
				fmt.Fprintf(w, "Text(%q)", tag.content)
			} else {
				fmt.Fprintf(w, "TextIf(%q, layout.%#v)", tag.content, t.cond)
			}
		case space:
			if t.cond == Always {
				fmt.Fprint(w, "Space()")
			} else {
				fmt.Fprintf(w, "SpaceIf(layout.%#v)", t.cond)
			}
		case newlines:
			if t.cond == Always {
				fmt.Fprintf(w, "Break(%d)", tag.count)
			} else {
				fmt.Fprintf(w, "BreakIf(%d, layout.%#v)", tag.count, t.cond)
			}
		}
		first = false
	}
}

// A condition determines when content added with the *If methods should be rendered.
type condition int

const (
	// Always renders the content unconditionally.
	Always condition = iota

	// Flat renders the content only when the containing group fits on a single line.
	Flat

	// Broken renders the content only when the containing group is broken across multiple lines.
	Broken
)

func (c condition) String() string {
	switch c {
	case Always:
		return "always"
	case Flat:
		return "flat"
	case Broken:
		return "broken"
	default:
		panic("condition string not implemented")
	}
}

func (c condition) GoString() string {
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

type node struct {
	tag     tag
	len     int
	cond    condition
	measure *measure
}

func (t *node) String() string {
	return fmt.Sprintf("Node{tag=%s, len=%d, cond=%s, measure=%s}", t.tag, t.len, t.cond, t.measure)
}

// measure represents the calculated width of a tag sequence during the measurement phase.
//
// A space is "trailing" if there's no content after it before the end of a sequence (or a
// break). The algorithm defers counting spaces until we know if they're trailing.
//
// Invariant: At any point, measure represents:
//   - width: definite width of non-trailing content
//   - pendingSpace: whether we have a space pending inclusion in width (if followed by content)
//   - broken: whether this sequence contains unconditional breaks
type measure struct {
	width        int
	broken       bool
	pendingSpace bool
}

func (m *measure) add(b measure) {
	if m.broken || b.broken {
		m.broken = true
		m.pendingSpace = false
	} else {
		// If b has content (width > 0) or has a pending space,
		// then our pending space gets included in width
		if b.width > 0 || b.pendingSpace {
			if m.pendingSpace {
				m.width++ // include pending space in width
			}
			m.pendingSpace = b.pendingSpace
		}
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
