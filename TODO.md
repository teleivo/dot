* improve lexing of quoted strings
  * test not escaping a quote?

* improve lexing of edge operators?
  * test invalid edge operators?

* test that I consume until the appropriate separators like a terminal, eof (this is probably
covered in every test), whitespace. Does the separator depend on the specific type of id?
    * for numeral
    * quoted should only be closing quote
    * unquoted id until I find a terminal or eof

* generate coverage to see if I missed any logic?

* refactor and fix todos in code

* what are not io.EOF errors and do I handle them well?

* support comments (by discarding them)

* handle EOF differently? I now have multiple places checking for io.EOF would be nice
  to mark that in one place
* how to continue generating tokens when finding invalid ones?

* write parser

* how to handle ports?

* write cmd/validate
* write cmd/stats that tells me how many nodes, edges there are

## Questions

* should I strip the quotes from the literal? or leave that up to the parser?

## Nice to have

* lex html string
* expose the knowledge of quoted, unquoted, numeral, html identifiers? 

### Hints

* "\n\n\n\t  - F" leads to "a numeral must have at least one digit" pointing to the whitespace
following the -. Is that understandable enough?
* add hints to some errors like <- did you mean ->
* non-breaking space between numerals leads to

echo 'graph{ 100 200 }' | dot -Tsvg -O
Warning: syntax ambiguity - badly delimited number '100' in line 1 of <stdin> splits into two tokens

