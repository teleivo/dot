* what does Break(0) mean? should I support this?
* tests
  * only test this as part of dotfmt or test it in isolation?
* measure in original sets broken if text contains newline. this is not correct for raw strings
right? `foo\nfaa` in Go or similar with escaped newlines or so in DOT should not cause a newline.
add a new tag/attribute? rawtext, `<text raw/>` or don't implement that?
* add indent
* add godocs
