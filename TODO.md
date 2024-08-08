* refactor and fix todos in code
* handle EOF differently? I now have multiple places checking for io.EOF would be nice
  to mark that in one place

* what are not io.EOF errors and do I handle them well?
* generate coverage to see if I missed any logic?

* how to continue generating tokens when finding invalid ones?

* write parser

* write cmd/validate
* write cmd/stats that tells me how many nodes, edges there are
* profile any of the above on a large file, generate a pprof dot file and feed that back into the
parser as a test via testdata

## Parser

* Where are commas legal?
* Are `{}` creating a lexical scope? This

```
{ node [shape=circle]
    a b c d e f g h  i j k l m n o p  q r s t u v w x
}
{ node [shape=diamond]
    A B C D E F G H  I J K L M N O P  Q R S T U V W X
}
```

sets the attributes on given nodes in the `{}` but will it affect nodes outside?

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

