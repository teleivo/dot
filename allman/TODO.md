* try smallest allman style layouting
* uint vs int
* what API do I expose? what would I like to use when mapping from dot nodes to this
  * the fluent API reads nicely but is there any trouble with my unexported types and the functions
    taking the prime names? would I put this into its own package (interal)? how would that change
  readability
  * add Group function like Break if the above makes sense
* pointers
* measure in original sets broken if text contains newline. this is not correct for raw strings
right? `foo\nfaa` in Go or similar with escaped newlines or so in DOT should not cause a newline.
add a new tag/attribute? rawtext, `<text raw/>` or don't implement that?
* add indent
