* rearrange identifier tests to the ones I use for numerals?
* test invalid edge operators?
* test more invalid identifiers, how does any string not leading with a digit concern
  lexing?
* lex html string
* generate coverage to see if I missed any logic?
* refactor and fix todos in code
* handle EOF differently? I now have multiple places checking for io.EOF would be nice
  to mark that in one place

* should I strip the quotes from the literal? or leave that up to the parser?
* should I expose the knowledge of quoted, unquoted, numeral, html identifiers? 
* add hints to some errors like <- did you mean ->
