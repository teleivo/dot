* write parser
  * parse node stmt
  * parse multiple statements
  * parse edge stmt
  * parse attribute stmt
  * parse subgraph
  * what is `ID '=' ID`
  * parse ports?

I want to be able to at least parse what I need for my current test setup

```
strict digraph {
	3 -> 2 [label="L"]
	5 -> 3 [label="L"]
	3 -> 4 [label="R"]
	10 -> 5 [label="L", color = red]
	7 -> 6 [label="L"]
	5 -> 7 [label="R"]
	9 -> 8 [label="L", color = red]
	7 -> 9 [label="R"]
	20 -> 15 [label="L"]
	10 -> 20 [label="R"]
	20 -> 23 [label="R"]
}
```

* how to continue generating tokens when finding invalid ones?

* count opening braces and brackets and decrement them on closing to validate they match?
or is that to simplistic as there are rules as to when you are allowed/have to close them?

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

### Compatibility & Fault Tolerance

This does stop at the first error

```
echo 'graph{ !A; C->B }' | dot -Tsvg -O
Error: <stdin>: syntax error in line 1 near '!'
```

and is not precise about where the error is

```
echo 'graph{ C->B; @A }' | dot -Tsvg -O
Error: <stdin>: syntax error in line 1 near '->'
```

Null byte is not allowed in unquoted identifiers as per spec. It is also not supported in quoted
strings as shown by this error

```
echo -e 'graph{ "A\000--B" }' | dot -Tsvg -O
Error: <stdin>: syntax error in line 1 scanning a quoted string (missing endquote? longer than 16384?)
String starting:"A
```

## Serializer

* serialize a dot.Graph given a writer
* test Parser/Serializer by feeding one to the other which should give the same result

## Questions

* should I strip the quotes from the literal? or leave that up to the parser?

## Nice to have


* lex html string
* expose the knowledge of quoted, unquoted, numeral, html identifiers? 
* how complicated is it to use the bufio.Readers buffer instead of creating intermediate slices for
identifiers? how much would that even matter at the expense of how much code :sweat_smile:

### Hints

* "\n\n\n\t  - F" leads to "a numeral must have at least one digit" pointing to the whitespace
following the -. Is that understandable enough?
* add hints to some errors like <- did you mean ->
* non-breaking space between numerals leads to

echo 'graph{ 100 200 }' | dot -Tsvg -O
Warning: syntax ambiguity - badly delimited number '100' in line 1 of <stdin> splits into two tokens

