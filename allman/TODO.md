* add indent
* what does Break(0) mean? should I support this?
* how to indent using tabs vs spaces? make this a fixed decision but in theory configurable on the
doc like NewDoc or so?
* tests
  * only test this as part of dotfmt or test it in isolation?
* measure in original sets broken if text contains newline. this is not correct for raw strings
right? `foo\nfaa` in Go or similar with escaped newlines or so in DOT should not cause a newline.
add a new tag/attribute? rawtext, `<text raw/>` or don't implement that?
* add godocs
