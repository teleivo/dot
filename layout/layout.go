// Package layout TODO
package layout

import (
	"fmt"
	"io"
	"strings"
)

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
	return d.tag(Text(content))
}

func (d *Doc) TextIf(content string, cond condition) *Doc {
	return d.tagIf(Text(content), cond)
}

func (d *Doc) Space() *Doc {
	return d.tag(Space)
}

func (d *Doc) SpaceIf(cond condition) *Doc {
	return d.tagIf(Space, cond)
}

func (d *Doc) Break(count int) *Doc {
	return d.tag(Break(count))
}

func (d *Doc) BreakIf(count int, cond condition) *Doc {
	return d.tagIf(Break(count), cond)
}

func (d *Doc) Group(body func(*Doc)) *Doc {
	return d.tagWith(&group{}, body)
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

func (d *Doc) Render(w io.Writer) {
	d.measure()
	d.layout(d.All(), 0)
	render(w, d.All(), true)
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
		parent.measure.Add(sumWidths(t, children))
	}
	return *parent.measure
}

func (d *Doc) layout(iter tagIterator, column int) {
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *group:
			if t.measure.width > d.maxColumn {
				t.measure.broken = true
			}
			d.layout(children, column)
		case *text:
			column += len(tag.content)
		case space:
			column++
		case newlines:
			// TODO reset width except 0 newlines?
			column = 0
		}
	}
}

func render(w io.Writer, iter tagIterator, isParentBroken bool) {
	for t, children := range iter {
		if t.cond == Flat && isParentBroken || t.cond == Broken && !isParentBroken {
			continue
		}

		switch tag := t.tag.(type) {
		case *group:
			render(w, children, t.measure.broken)
		case *text:
			fmt.Fprintf(w, "%s", tag.content)
		case space:
			fmt.Fprintf(w, " ")
		case newlines:
			// TODO is batching prints more efficient? like having a slice of 10 newlines and
			// printing at least up to 10 at a time?
			for i := tag.count; i > 0; i-- {
				fmt.Fprintf(w, "\n")
			}
		}
	}
}

func (d *Doc) String() string {
	var sb strings.Builder
	stringIter(&sb, d.All())
	return sb.String()
}

func stringIter(w io.Writer, iter tagIterator) {
	// TODO when to print newlines even in this debug string?
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *group:
			fmt.Fprintf(w, "<group width=%s>", t.measure)
			stringIter(w, children)
			fmt.Fprintf(w, "</group>")
		case *text:
			fmt.Fprintf(w, "<text width=%s content=%q/>", t.measure, tag.content)
		case space:
			fmt.Fprintf(w, "<space/>")
		case newlines:
			fmt.Fprintf(w, "<break count=%d/>", tag.count)
		}
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

func (m *measure) Add(b measure) {
	if m.broken || b.broken {
		m.broken = true
	} else {
		m.width += b.width
	}
}

func (m *measure) IsBroken() bool {
	return m.broken
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

type group struct{}

// Group a sequence of tags to be rendered as one line or multiple lines if they exceed the maximum
// column.
func Group() *group {
	return &group{}
}

func (g *group) tag() {}

func (g *group) String() string {
	// TODO implement
	return "Group"
}

type text struct {
	content string
}

func Text(content string) *text {
	return &text{content: content}
}

func (t *text) tag() {}

func (t *text) String() string {
	return fmt.Sprintf("Text(%q)", t.content)
}

var Space = space{}

type space struct{}

func (s space) tag() {}

func (s space) String() string {
	return "Space"
}

type newlines struct {
	count int
}

func Break(count int) newlines {
	return newlines{count: count}
}

func (n newlines) tag() {}

func (n newlines) String() string {
	return fmt.Sprintf("Break(%d)", n.count)
}
