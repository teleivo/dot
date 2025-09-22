package main

import (
	"fmt"
	"io"
	"os"
)

// ! // flat: fn foo() { ... }
// ! //
// ! // broken:
// ! // fn foo()
// ! // {
// ! //   // ...
// ! // }
// ! Doc::new()
// !   .tag("fn")
// !   .tag(Tag::Space)
// !   .tag("foo")
// !   .tag("(").tag(")")
// !   .tag_with(Tag::Group(40), |doc| {
// !     doc
// !       .tag_if(Tag::Space, If::Flat)
// !       .tag_if(Tag::Break(1), If::Broken)
// !       .tag("{")
// !       .tag_if(Tag::Space, If::Flat)
// !       .tag_if(Tag::Break(1), If::Broken)
// !       .tag_with(Tag::Indent(2), |doc| {
// !         // Brace contents here...
// !       })
// !       .tag_if(Tag::Space, If::Flat)
// !       .tag_if(Tag::Break(1), If::Broken)
// !       .tag("}");
// !   });
// ! ```
func main() {
	d := New().
		Tag(Text("package main")).
		Tag(Break(1)).
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
	tags []TagInfo
}

func New() *Doc {
	return &Doc{}
}

type TagIterator func(yield func(TagInfo, TagIterator) bool)

func (d *Doc) All() TagIterator {
	return d.newTagIterator(0, uint(len(d.tags)))
}

func (d *Doc) newTagIterator(i, j uint) TagIterator {
	return func(yield func(TagInfo, TagIterator) bool) {
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
	d.tags = append(d.tags, TagInfo{tag: t, len: 0, cond: cond})
	body(d)
	if j := uint(len(d.tags)); j != i {
		d.tags[i].len = j - i - 1
	}
	return d
}

func (d *Doc) Render(w io.Writer) {
	renderIter(w, d.All())
}

func renderIter(w io.Writer, iter TagIterator) {
	for t, children := range iter {
		switch tag := t.tag.(type) {
		case *Group:
			fmt.Fprintf(w, "<group>")
			renderIter(w, children)
			fmt.Fprintf(w, "</group>")
		case *text:
			fmt.Fprintf(w, "<text content=%q/>", tag.content)
		case space:
			fmt.Fprintf(w, "<space/>")
		case newlines:
			fmt.Fprintf(w, "<break count=%d/>", tag.count)
		}
	}
	// TODO measure
	// TODO layout
	// TODO print
}

type condition int

const (
	Always condition = iota
	Flat
	Broken
)

// TODO what is the benefit of wrapping Tag? is it so a Tag is the API and users cannot mess with
// measurement and len? can I achieve that without yet another type

type TagInfo struct {
	tag  Tag
	len  uint
	cond condition
	// measure
}

type Tag interface {
	tag()
}

type Group struct{}

func (g *Group) tag() {}

type text struct {
	content string
}

func Text(content string) *text {
	return &text{content}
}

func (t *text) tag() {}

var Space = space{}

type space struct{}

func (s space) tag() {}

type newlines struct {
	count uint
}

func Break(count uint) newlines {
	return newlines{count}
}

func (n newlines) tag() {}
