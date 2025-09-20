package main

import (
	"io"
	"os"
)

//! // flat: fn foo() { ... }
//! //
//! // broken:
//! // fn foo()
//! // {
//! //   // ...
//! // }
//! Doc::new()
//!   .tag("fn")
//!   .tag(Tag::Space)
//!   .tag("foo")
//!   .tag("(").tag(")")
//!   .tag_with(Tag::Group(40), |doc| {
//!     doc
//!       .tag_if(Tag::Space, If::Flat)
//!       .tag_if(Tag::Break(1), If::Broken)
//!       .tag("{")
//!       .tag_if(Tag::Space, If::Flat)
//!       .tag_if(Tag::Break(1), If::Broken)
//!       .tag_with(Tag::Indent(2), |doc| {
//!         // Brace contents here...
//!       })
//!       .tag_if(Tag::Space, If::Flat)
//!       .tag_if(Tag::Break(1), If::Broken)
//!       .tag("}");
//!   });
//! ```
func main() {
	d:= New().
		Tag(Text("package main")).
		Tag(Break(1)).
		Tag(Text("func")).
		Tag(Space).
		Tag(Text("main")).
		Tag(Text("(")).Tag(Text(")")).
		TagWith(&Group{}, func(d *Doc) {
			d.
				TagIf(Space, flat).
				TagIf(Break(1), broken).
				Tag(Text("{")).
				TagIf(Space, flat).
				TagIf(Break(1), broken).
				Tag(Text(`print("yes")`)).
				TagIf(Space, flat).
				TagIf(Break(1), broken).
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

func (d *Doc) Tag(t Tag) (*Doc) {
	return d.tagIfWith(t, always, nil)
}

func (d *Doc) TagIf(t Tag, cond condition) (*Doc) {
	return d.tagIfWith(t, cond, nil)
}

func (d *Doc) TagWith(t Tag, body func(*Doc)) (*Doc) {
	return d.tagIfWith(t, always, body)
}

func (d *Doc) tagIfWith(t Tag, cond condition, body func(*Doc)) (*Doc) {
	// TODO handle len and then walk tree
	d.tags=append(d.tags, TagInfo{tag: t, len: 0, cond: cond})
	body(d)
	return d
}

func (d *Doc) Render(w io.Writer) {
	// TODO walk tree
// TODO measure
// TODO layout
// TODO print
}

type condition int

const (
	always condition = iota
	flat
	broken
)

type TagInfo struct {
	tag Tag
	len uint
	cond condition
	// measure
}

type Tag interface {
  tag()
}

type Group struct {}

func (g *Group) tag() {}

type text struct {
	content string
}

func Text(content string) *text {
	return &text{content}
}

func (t *text) tag() {}

var	Space = space{}

type space struct {}

func (s space) tag() {}

type newlines struct {
	count uint
}

func Break(count uint) newlines {
	return newlines{count}
}

func (n newlines) tag() {}

