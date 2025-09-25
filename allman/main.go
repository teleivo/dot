package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	d := New().
		Tag(Text("package main")).
		Tag(Break(2)).
		Tag(Text("func")).
		Tag(Space).
		Tag(Text("main")).
		Tag(Text("(")).Tag(Text(")")).
		TagWith(&Group{}, func(d *Doc) {
			d.
				TagIf(Space, Flat).
				TagIf(Break(1), Broken).
				Tag(Text("{")).
				TagIf(Space, Flat).
				TagIf(Break(1), Broken).
				Tag(Text(`print("yes")`)).
				TagIf(Space, Flat).
				TagIf(Break(1), Broken).
				Tag(Text("}"))
		})
	d.Render(os.Stdout)
}

type Doc struct {
	tags []*TagInfo
}

func New() *Doc {
	return &Doc{}
}

type TagIterator func(yield func(*TagInfo, TagIterator) bool)

func (d *Doc) All() TagIterator {
	return d.newTagIterator(0, uint(len(d.tags)))
}

func (d *Doc) newTagIterator(i, j uint) TagIterator {
	return func(yield func(*TagInfo, TagIterator) bool) {
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

func (d *Doc) Tag(t Tag) *Doc {
	return d.tagIfWith(t, Always, func(d *Doc) {})
}

func (d *Doc) TagIf(t Tag, cond condition) *Doc {
	return d.tagIfWith(t, cond, func(d *Doc) {})
}

func (d *Doc) TagWith(t Tag, body func(*Doc)) *Doc {
	return d.tagIfWith(t, Always, body)
}

func (d *Doc) tagIfWith(t Tag, cond condition, body func(*Doc)) *Doc {
	i := uint(len(d.tags))
	d.tags = append(d.tags, &TagInfo{tag: t, len: 0, cond: cond, measure: &Measure{}})
	body(d)
	if j := uint(len(d.tags)); j != i {
		d.tags[i].len = j - i - 1
	}
	return d
}

func (d *Doc) Render(w io.Writer) {
	d.measure()
	// TODO layout
	// TODO implement actual rendering
	// fmt.Println(d.DebugString())
	renderIter(w, d.All(), true)
}

func (d *Doc) measure() {
	for t, children := range d.All() {
		measure(t, children)
	}
	for t, children := range d.All() {
		sumWidths(t, children)
	}
}

func measure(parent *TagInfo, children TagIterator) {
	tagWidth(parent)
	for t, children := range children {
		measure(t, children)
	}
}

func tagWidth(t *TagInfo) {
	if t.cond == Broken { // only measure flat width
		return
	}

	switch tag := t.tag.(type) {
	case *text:
		t.measure.width = uint(len(tag.content))
	case space:
		t.measure.width = 1
	case newlines:
		t.measure.broken = true
	}
}

func sumWidths(parent *TagInfo, children TagIterator) Measure {
	for t, children := range children {
		parent.measure.Add(sumWidths(t, children))
	}
	return *parent.measure
}

func renderIter(w io.Writer, iter TagIterator, isParentBroken bool) {
	for t, children := range iter {
		if t.cond == Flat && isParentBroken || t.cond == Broken && !isParentBroken {
			continue
		}

		switch tag := t.tag.(type) {
		case *Group:
			renderIter(w, children, t.measure.broken)
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

// TODO make this the normal String()?

func (d *Doc) DebugString() string {
	var sb strings.Builder
	debugString(&sb, d.All())
	return sb.String()
}

func debugString(w io.Writer, iter TagIterator) {
	// TODO when to print newlines even in this debug string?
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *Group:
			fmt.Fprintf(w, "<group width=%s>", t.measure)
			debugString(w, children)
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

type TagInfo struct {
	tag     Tag
	len     uint
	cond    condition
	measure *Measure
}

func (t *TagInfo) String() string {
	return fmt.Sprintf("TagInfo{tag=%s, len=%d, cond=%s, measure=%s}", t.tag, t.len, t.cond, t.measure)
}

type Measure struct {
	width  uint
	broken bool
}

func (m *Measure) Add(b Measure) {
	if m.broken || b.broken {
		m.broken = true
	} else {
		m.width += b.width
	}
}

func (m *Measure) IsBroken() bool {
	return m.broken
}

func (m *Measure) String() string {
	if m.broken {
		return "broken"
	}
	return fmt.Sprint(m.width)
}

type Tag interface {
	tag()
}

type Group struct{}

func (g *Group) tag() {}

func (g *Group) String() string {
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
	count uint
}

func Break(count uint) newlines {
	return newlines{count: count}
}

func (n newlines) tag() {}

func (n newlines) String() string {
	return fmt.Sprintf("Break(%d)", n.count)
}
