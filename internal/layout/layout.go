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
	"bufio"
	"fmt"
	"io"
	"math"
	"strings"
)

// Format specifies the output representation for rendering a [Doc].
type Format int

const (
	// Default renders the formatted output as text.
	Default Format = iota
	// Layout renders the document structure using HTML-like syntax, showing all nodes including
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

var validFormats = [...]string{"default", "go", "layout"}

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
	nodes     []node
}

// NewDoc creates a new document with the specified maximum column width. Text will be reflowed
// to fit within this width where possible.
func NewDoc(maxColumn int) *Doc {
	return &Doc{maxColumn: maxColumn}
}

// HasTrailingSpace reports whether the last tag added to the document is whitespace (space or break).
func (d *Doc) HasTrailingSpace() bool {
	if len(d.nodes) == 0 {
		return false
	}
	k := d.nodes[len(d.nodes)-1].kind
	return k == spaceTag || k == newlineTag
}

// Clone creates a deep copy of the Doc. Use this if you want to [Doc.Render] a Doc multiple times.
func (d *Doc) Clone() *Doc {
	clone := &Doc{
		maxColumn: d.maxColumn,
		nodes:     make([]node, len(d.nodes)),
	}
	for i, t := range d.nodes {
		clone.nodes[i] = node{
			kind:    t.kind,
			content: t.content,
			count:   t.count,
			len:     t.len,
			cond:    t.cond,
		}
	}
	return clone
}

type nodeRange struct {
	start, end int
}

// All returns a nodeRange over all nodes in the document.
func (d *Doc) All() nodeRange {
	return nodeRange{0, len(d.nodes)}
}

// Text adds literal text content to the document.
func (d *Doc) Text(content string) *Doc {
	return d.addNode(node{kind: textTag, content: content}, Always, func(d *Doc) {})
}

// TextIf adds literal text content that only renders when the specified condition is met.
func (d *Doc) TextIf(content string, cond condition) *Doc {
	return d.addNode(node{kind: textTag, content: content}, cond, func(d *Doc) {})
}

// Space adds a single space to the document.
func (d *Doc) Space() *Doc {
	return d.addNode(node{kind: spaceTag}, Always, func(d *Doc) {})
}

// SpaceIf adds a single space that only renders when the specified condition is met.
func (d *Doc) SpaceIf(cond condition) *Doc {
	return d.addNode(node{kind: spaceTag}, cond, func(d *Doc) {})
}

// Break adds one or more newlines to the document. The count must be positive.
func (d *Doc) Break(count int) *Doc {
	if count <= 0 {
		panic("Break: count must be positive")
	}
	return d.addNode(node{kind: newlineTag, count: count}, Always, func(d *Doc) {})
}

// BreakIf adds one or more newlines that only render when the specified condition is met.
// The count must be positive.
func (d *Doc) BreakIf(count int, cond condition) *Doc {
	if count <= 0 {
		panic("BreakIf: count must be positive")
	}
	return d.addNode(node{kind: newlineTag, count: count}, cond, func(d *Doc) {})
}

// Group marks a sequence of content that should be kept on one line if it fits within the
// maximum column width, or broken across multiple lines if it doesn't.
func (d *Doc) Group(body func(*Doc)) *Doc {
	return d.addNode(node{kind: groupTag}, Always, body)
}

// Indent increases the indentation level by the specified number of columns for the content
// added in body. The indentation is applied at the start of each line after a newline.
// Each column of indentation is rendered as a single tab character.
func (d *Doc) Indent(columns int, body func(*Doc)) *Doc {
	return d.addNode(node{kind: indentTag, count: columns}, Always, body)
}

func (d *Doc) addNode(n node, cond condition, body func(*Doc)) *Doc {
	i := len(d.nodes)

	// merge consecutive spaces of the same condition
	if n.kind == spaceTag && i > 0 {
		if d.nodes[i-1].kind == spaceTag && cond == d.nodes[i-1].cond {
			return d
		}
	}

	n.cond = cond
	d.nodes = append(d.nodes, n)
	body(d)
	if j := len(d.nodes); j != i {
		d.nodes[i].len = j - i - 1
	}
	return d
}

// Render writes the formatted document to the writer in the specified format. Note that rendering
// mutates the document, so re-rendering the same document will produce incorrect results. Use
// [Doc.Clone] to create a copy if you need to render multiple times or to different outputs.
func (d *Doc) Render(w io.Writer, format Format) error {
	d.measure()
	d.layout(d.All(), 0, 0)
	bw := bufio.NewWriter(w)
	r := &renderer{nodes: d.nodes, w: bw}

	var err error
	switch format {
	case Default:
		err = r.render(d.All(), true)
	case Layout:
		_, err = fmt.Fprint(bw, d)
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
		_, err = fmt.Fprintf(bw, goTemplate, goString(d, 1))
	}

	if err != nil {
		return err
	}
	return bw.Flush()
}

type renderer struct {
	nodes           []node    // nodes is the flat node slice from the Doc
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
	all := d.All()
	d.measureIter(all)
	d.sumWidths(all)
}

func (d *Doc) measureIter(nr nodeRange) {
	for i := nr.start; i < nr.end; {
		t := &d.nodes[i]
		tagWidth(t)
		if t.len > 0 {
			children := nodeRange{i + 1, i + 1 + t.len}
			d.measureIter(children)
			i = i + 1 + t.len
		} else {
			i++
		}
	}
}

func tagWidth(t *node) {
	if t.cond == Broken { // only measure flat width
		return
	}

	switch t.kind {
	case textTag:
		t.measure.width = len(t.content)
	case spaceTag:
		// Spaces start as pending - they'll be included in width during sumWidths if
		// followed by content
		t.measure.pendingSpace = true
	case newlineTag:
		t.measure.broken = true
	}
}

func (d *Doc) sumWidths(nr nodeRange) {
	for i := nr.start; i < nr.end; {
		t := &d.nodes[i]
		if t.len > 0 {
			children := nodeRange{i + 1, i + 1 + t.len}
			d.sumWidths(children)
			// sum children's measures into parent
			for j := children.start; j < children.end; {
				child := &d.nodes[j]
				t.measure.add(child.measure)
				if child.len > 0 {
					j = j + 1 + child.len
				} else {
					j++
				}
			}
			i = i + 1 + t.len
		} else {
			i++
		}
	}
}

func (d *Doc) layout(nr nodeRange, indent, column int) {
	for i := nr.start; i < nr.end; {
		t := &d.nodes[i]
		switch t.kind {
		case groupTag:
			if t.measure.broken || column+t.measure.width > d.maxColumn {
				t.measure.broken = true
				children := nodeRange{i + 1, i + 1 + t.len}
				d.layout(children, indent, column)
			} else {
				column += t.measure.width
			}
		case indentTag:
			children := nodeRange{i + 1, i + 1 + t.len}
			d.layout(children, safeAdd(indent, t.count), column)
		case textTag:
			column += len(t.content)
		case spaceTag:
			column++
		case newlineTag:
			column = indent
		}
		if t.len > 0 {
			i = i + 1 + t.len
		} else {
			i++
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

func (r *renderer) render(nr nodeRange, isParentBroken bool) error {
	for i := nr.start; i < nr.end; {
		t := r.nodes[i]
		if t.cond == Flat && isParentBroken || t.cond == Broken && !isParentBroken {
			if t.len > 0 {
				i = i + 1 + t.len
			} else {
				i++
			}
			continue
		}

		switch t.kind {
		case groupTag:
			children := nodeRange{i + 1, i + 1 + t.len}
			if err := r.render(children, t.measure.broken); err != nil {
				return err
			}
		case indentTag:
			children := nodeRange{i + 1, i + 1 + t.len}
			r.indent = safeAdd(r.indent, t.count)
			if err := r.render(children, isParentBroken); err != nil {
				return err
			}
			r.indent -= t.count
		case textTag:
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
			if err := r.write(t.content); err != nil {
				return err
			}
			r.writtenNewlines = 0 // reset newlines as text means we do not deal with consecutive newlines
		case spaceTag:
			r.pendingSpace = true // writing space is delayed as it might be trailing
		case newlineTag:
			r.pendingSpace = false // discard pending space which would be trailing
			// merge consecutive Breaks
			for ; r.writtenNewlines < t.count; r.writtenNewlines++ {
				if err := r.write("\n"); err != nil {
					return err
				}
			}
		}
		if t.len > 0 {
			i = i + 1 + t.len
		} else {
			i++
		}
	}
	return nil
}

// String returns the document structure as HTML-like markup, showing all nodes and their properties.
// This implements [fmt.Stringer] and is like rendering with [Layout] format except that the measure
// and layout phases are not run. Useful for debugging the layout algorithm.
func (d *Doc) String() string {
	var sb strings.Builder
	d.stringIter(&sb, d.All(), 0)
	return sb.String()
}

func (d *Doc) stringIter(w *strings.Builder, nr nodeRange, indent int) {
	for i := nr.start; i < nr.end; {
		t := d.nodes[i]
		switch t.kind {
		case groupTag:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<group width=%s>\n", t.measure)
			children := nodeRange{i + 1, i + 1 + t.len}
			d.stringIter(w, children, indent+1)
			writeIndent(w, indent)
			fmt.Fprintf(w, "</group>\n")
		case indentTag:
			writeIndent(w, indent)
			fmt.Fprintf(w, "<indent columns=%d>\n", t.count)
			children := nodeRange{i + 1, i + 1 + t.len}
			d.stringIter(w, children, indent+1)
			writeIndent(w, indent)
			fmt.Fprintf(w, "</indent>\n")
		case textTag:
			writeIndent(w, indent)
			switch t.cond { // width is not computed for text that only renders when layout is Broken
			case Always:
				fmt.Fprintf(w, "<text width=%s content=%q/>\n", t.measure, t.content)
			case Flat:
				fmt.Fprintf(w, "<text cond=%q width=%s content=%q/>\n", t.cond, t.measure, t.content)
			default:
				fmt.Fprintf(w, "<text cond=%q content=%q/>\n", t.cond, t.content)
			}
		case spaceTag:
			writeIndent(w, indent)
			if t.cond == Always {
				fmt.Fprintf(w, "<space/>\n")
			} else {
				fmt.Fprintf(w, "<space cond=%q/>\n", t.cond)
			}
		case newlineTag:
			writeIndent(w, indent)
			if t.cond == Always {
				fmt.Fprintf(w, "<break count=%d/>\n", t.count)
			} else {
				fmt.Fprintf(w, "<break cond=%q count=%d/>\n", t.cond, t.count)
			}
		}
		if t.len > 0 {
			i = i + 1 + t.len
		} else {
			i++
		}
	}
}

func writeIndent(w *strings.Builder, columns int) {
	for range columns {
		w.WriteByte('\t')
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
	_, _ = fmt.Fprintf(&sb, "layout.NewDoc(%d)\n", d.maxColumn)
	d.goStringIter(&sb, d.All(), indent)
	return sb.String()
}

func (d *Doc) goStringIter(w *strings.Builder, nr nodeRange, indent int) {
	first := true
	for i := nr.start; i < nr.end; {
		t := d.nodes[i]
		if first {
			writeIndent(w, indent)
			fmt.Fprint(w, "d.\n")
			indent++
		} else {
			fmt.Fprint(w, ".\n")
		}
		writeIndent(w, indent)

		switch t.kind {
		case groupTag:
			children := nodeRange{i + 1, i + 1 + t.len}
			fmt.Fprint(w, "Group(func(d *layout.Doc) {\n")
			d.goStringIter(w, children, indent+1)
			fmt.Fprintln(w)
			writeIndent(w, indent)
			fmt.Fprintf(w, "})")
		case indentTag:
			children := nodeRange{i + 1, i + 1 + t.len}
			fmt.Fprintf(w, "Indent(%d, func(d *layout.Doc) {\n", t.count)
			d.goStringIter(w, children, indent+1)
			fmt.Fprintln(w)
			writeIndent(w, indent)
			fmt.Fprint(w, "})")
		case textTag:
			if t.cond == Always {
				fmt.Fprintf(w, "Text(%q)", t.content)
			} else {
				fmt.Fprintf(w, "TextIf(%q, layout.%#v)", t.content, t.cond)
			}
		case spaceTag:
			if t.cond == Always {
				fmt.Fprint(w, "Space()")
			} else {
				fmt.Fprintf(w, "SpaceIf(layout.%#v)", t.cond)
			}
		case newlineTag:
			if t.cond == Always {
				fmt.Fprintf(w, "Break(%d)", t.count)
			} else {
				fmt.Fprintf(w, "BreakIf(%d, layout.%#v)", t.count, t.cond)
			}
		}
		first = false
		if t.len > 0 {
			i = i + 1 + t.len
		} else {
			i++
		}
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

type tagKind int

const (
	textTag tagKind = iota
	spaceTag
	newlineTag
	groupTag
	indentTag
)

type node struct {
	kind    tagKind
	content string // text content
	count   int    // newline count or indent columns
	len     int
	cond    condition
	measure measure
}

func (t *node) String() string {
	return fmt.Sprintf("Node{kind=%s, len=%d, cond=%s, measure=%s}", t.kind, t.len, t.cond, t.measure)
}

func (k tagKind) String() string {
	switch k {
	case textTag:
		return "text"
	case spaceTag:
		return "space"
	case newlineTag:
		return "newline"
	case groupTag:
		return "group"
	case indentTag:
		return "indent"
	default:
		panic(fmt.Sprintf("unknown tagKind: %d", k))
	}
}

// measure represents the calculated width of a node sequence during the measurement phase.
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

func (m measure) String() string {
	if m.broken {
		return "broken"
	}
	return fmt.Sprint(m.width)
}
