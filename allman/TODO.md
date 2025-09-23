* try smallest allman style layouting
* uint vs int
* what API do I expose? what would I like to use when mapping from dot nodes to this
* pointers
* measure in original sets broken if text contains newline. this is not correct for raw strings
right? `foo\nfaa` in Go or similar with escaped newlines or so in DOT should not cause a newline.
add a new tag/attribute? rawtext, `<text raw/>` or don't implement that?
