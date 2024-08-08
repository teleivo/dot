* test invalid edge operators?
* test that I validate every rune I add to the id of that ids type
* test that I read a numeral, unquoted id until I find a terminal or eof
* test that I read a quoted id until I find a terminal or eof or a non escaped quote

* test more invalid identifiers, how does any string not leading with a digit concern
  lexing?

* how to continue generating tokens when finding invalid ones?

* generate coverage to see if I missed any logic?

* refactor and fix todos in code

* handle EOF differently? I now have multiple places checking for io.EOF would be nice
  to mark that in one place

* write parser

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

