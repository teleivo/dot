* write cmd/dotfmt
    * try formatting all https://gitlab.com/graphviz/graphviz/-/tree/main/graphs?ref_type=heads
    any errors?
    * update README with an example

    * support comments
      * first the parser needs to parse comments anywhere. right now comments lead to errors in a
      lot of places they should be legal
      interestingly comments are ok on their own line or inside a subgraph
      ../graphviz/graphs/uncommented/honda-tokoro.gv
      `{/*L=m*/rank=same n001 n011}`
      the comment turns into an empty line right now
      ```dot
      	subgraph {

		rank=same
		n001
		n011
	}
      ```

* why is the ../graphviz/graphs/uncommented/russian.gv not stripping the leading whitespace from
  before graph?

* remove empty statements? like or does that serve any purpose?

```
graph [
];
```

* why is ../graphviz/graphs/uncommented/pgram.gv label ID not broken up?

```
	subgraph {
		rank=same
		node [shape=parallelogram]
		"Parallelogram" [label="This is a test\nof a multiline\nlabel in an\nparallelogram with approx\nsquare aspect"]
		"a ----- long thin parallelogram"
		"xx" [label="m"]
		"yy" [label="a\nb\nc\nd\ne\nf"]
		node [shape=octagon]
		"Octagon" [label="This is a test\nof a multiline\nlabel in an\noctagon with approx\nsquare aspect"]
		node [shape=parallelogram]
		"Parallelogram" [label="This is a test\nof a multiline\nlabel in an\nparallelogram with approx\nsquare aspect"]
		"a ----- long thin parallelogram"
		"zz" [label="m"]
		"qq" [label="a\nb\nc\nd\ne\nf"]
		ordering=out
	}
```
    * allow multiple nodes on the same line. how to break them up when > maxCol

    * how to treat newlines? right now they are discarded. Maybe I'd like to group/make blocks.
    Allow users to do that. No more than one empty line though. And will that line be completely
    empty or be indented as the surrounding code?
    I need proper token/ast position. for this row and column

    * support parsing/formatting ranges

    * test parser/lexer with invalid ID as ID for port. check the places were convert literals to
    ast.ID without parsing the identifier, should I not parse it first?

* how to handle error on fmt.Fprint?
* how to handle errors?

* add section in readme or add own readme in ./cmd/dotfmt/?
  * indentation: tabs
  * alignment: spaces
    * every comment starts with one space only, extra whitespace is removed
  * max number of utf8 characters per line 100
    * only if the indentation is < than ???
    * IDs are broken up into multiple lines and quoted if they were not already quoted
    * comments are broken up into multiple lines using the same marker that was used

    ```dot
        # comment that is too long
    ```

    turns into

    ```dot
        # comment that
        # is too long
    ```

* add profiling flags
    * capture profile formatting example dot files
    * capture profiles formatting the profile dot file
    * all of this to find any lingering bugs I have
* try formatting invalid dot and improve error handling
  * `2->4` leads to error
  "2:15: a numeral can only be prefixed with a `-`"
  allow that :) and turn it into `2 -> 4`

improve
* handling of EOF better and move these special tokens up top like Go does

* count opening braces and brackets and decrement them on closing to validate they match?
or is that to simplistic as there are rules as to when you are allowed/have to close them?

* still needed? Reuse some of the tests later when I use the parser to evaluate the AST to the simpler Graph types

```go
type Graph struct {
	ID       string
	Strict   bool
	Directed bool
	Nodes    map[string]*Node
}

type Node struct {
	ID         string
	Attributes map[string]Attribute
}

type Attribute struct {
	Name, Value string
}

func TestParser(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want dot.Graph
			err  error
		}{
			"Empty": {
				in: "",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{},
				},
			},
			"EmptyDirectedGraph": {
				in: "digraph {}",
				want: dot.Graph{
					Directed: true,
					Nodes:    map[string]*dot.Node{},
				},
			},
			"EmptyUndirectedGraph": {
				in: "graph {}",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{},
				},
			},
			"StrictDirectedUnnamedGraph": {
				in: `strict digraph {}`,
				want: dot.Graph{
					Strict:   true,
					Directed: true,
					Nodes:    map[string]*dot.Node{},
				},
			},
			"StrictDirectedNamedGraph": {
				in: `strict digraph dependencies {}`,
				want: dot.Graph{
					Strict:   true,
					Directed: true,
					ID:       "dependencies",
					Nodes:    map[string]*dot.Node{},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"StrictMustBeFirstKeyword": {
					in:     "digraph strict {}",
					errMsg: `got "strict" instead`,
				},
				"GraphIDMustComeAfterGraphKeywords": {
					in:     "dependencies {}",
					errMsg: `got "dependencies" instead`,
				},
				"LeftBraceMustFollow": {
					in:     "graph dependencies [",
					errMsg: `got "[" instead`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.New(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})

	t.Run("NodeStatement", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want dot.Graph
			err  error
		}{
			"OnlyNode": {
				in: "graph { foo }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"OnlyNodes": {
				in: "graph { foo ; bar baz }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
						"bar": {ID: "bar", Attributes: map[string]dot.Attribute{}},
						"baz": {ID: "baz", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"OnlyNodeWithEmptyAttributeList": {
				in: "graph { foo [] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"NodeWithSingleAttribute": {
				in: "graph { foo [a=b] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingComma": {
				in: "graph { foo [a=b,] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingSemicolon": {
				in: "graph { foo [a=b;] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributeOverriding": {
				in: "graph { foo [a=b;c=d]; foo [a=e] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "e"},
								"c": {Name: "c", Value: "d"},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInSingleBracketPair": {
				in: "graph { foo [a=b c=d,e=f;g=h] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
								"c": {Name: "c", Value: "d"},
								"e": {Name: "e", Value: "f"},
								"g": {Name: "g", Value: "h"},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInMultipleBracketPairs": {
				in: "graph { foo [a=b c=d][e=f;g=h] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
								"c": {Name: "c", Value: "d"},
								"e": {Name: "e", Value: "f"},
								"g": {Name: "g", Value: "h"},
							},
						},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"AttributeListWithoutClosingBracket": {
					in:     "graph { foo [ }",
					errMsg: `expected next token to be one of ["]" "identifier"]`,
				},
				"AttributeWithoutName": {
					in:     "graph { foo [ = b ] }",
					errMsg: `expected next token to be one of ["]" "identifier"]`,
				},
				"AttributeWithoutValue": {
					in:     "graph { foo [ a = ] }",
					errMsg: `expected next token to be "identifier"`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.New(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})
}

func assertContains(t *testing.T, got, want string) {
	if !strings.Contains(got, want) {
		t.Errorf("got %q which does not contain %q", got, want)
	}
}
```

* write cmd/dothot hot-reloading a file passing it to dot and showing its svg in the browser
* write cmd/validate
* write cmd/stats that tells me how many nodes, edges there are
* profile any of the above on a large file, generate a pprof dot file and feed that back into the
parser as a test via testdata


## API

* should I add the token to the AttrStmt? so it is easier to check if its a graph/node/edge?
* is it nicer to work with slices then my choice of linked lists with *Next whenever there was a
recursive definition?
* should I remove the Directed field from EdgeRHS as that is clear from graph.Directed?
* make error messages more user friendly
  * for example when parsing the attr_stmt the attr_list is mandatory, instead of saying expected [
    I could say that

## Language Feature Support

* support concatenating strings?
https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
> In addition, double-quoted strings can be concatenated using a '+' operator.
* lex html string

## Parser

* Lexical and Semantic Notes https://graphviz.org/doc/info/lang.html
  * should some of these influence the parser/should it err
  * how does strict affect a graph? no cycles? is that something my parser should validate?
* how to continue generating tokens when finding invalid ones? create an invalid/illegal token? how
  does treesitter do it? they have a missing node and an illegal one?
* Add position start, end to tokens as in Gos' token package. Add them to ast/Node as well like Go
does? Their columns are bytes not runes, should I use bytes as well?
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

## dotfmt

* comments
    * should have one " " after the marker
    * break up > 100 runes keeping the type of comment. so // will get another // on the next
    line
* test using dot examples from gallery
https://gitlab.com/graphviz/graphviz/-/tree/main/graphs?ref_type=heads
* test using invalid input
  * invalid input should be printed as is, it should not delete user input!

* add profiling flags

* support formatting file/dirs in place
  * goroutines could be fun once its working ;)
  * format all of https://gitlab.com/graphviz/graphviz/-/tree/main/graphs?ref_type=heads
    * profile, anything obvious I could improve?
    * add a benchmark to ensure no regressions

  * gofumpt uses positional args as files and reads from stdin if non given
```go
    args := flag.Args()
    if len(args) == 0 {
```
* this is a hint on how gofumpt can format pieces of go
// If we are formatting stdin, we accept a program fragment in lieu of a
// complete source file.
fragmentOk = true

if tries `parser.ParseFile` and returns if there is no error
in case of an error it adds `package p;` and tries `parser.ParseFile` again
if that fails it assumes the src might be statements and wraps it in a package with a function and
tries to ParseFile again. It creates an adjustSrc func to adjust the src again afterwards. It also
uses `;` so line numbers stay correct.

I could also try parsing a Graph, if that fails due to an error in the parseHeader I could wrap it
in a `graph { }` assuming that the src is a []Stmt. This might fail if src contains directed edges
so I need to detect such errors and try with `digraph {}`.

### Features

only the parser has access to the lexemes
* align multiple attribute values (and `=`)
	`"0" -- "1" -- "2" -- "3" -- "4" -- "0" [
		color = "blue"
		len   = 2.6
	]`
 should that then apply to the entire file :joy:? as global attributes can be set on the
graph/subraph as well
* strip unnecessary quotes
  * unstripped ID would need to be a valid ID, imagine `"A->B"` quotes cannot be stripped here
  * is the "easiest" to try to parse the unquoted literal as ID and only if valid strip them
* keep the indentation when splitting to multiple lines?
  * the parser would need to support + so I can concatenat IDs
* maybe: support subraph shorthand using `{}` and don't always print `subgraph` by looking at the literal? might need to add that to the ast as

## Highl Level API

I would like to define dot graphs in Go without having to create an ast. Like

```go
dot.Graph{
    ID: "galaxy",
    Attributes: []dot.Attribute{
        dot.Attribute{
            Name: "",
            Value: "",
        }
    }
}
```

I then want to print that to `io.Writer` in dot format. I could achieve that by going from the above
to an `ast.Graph` then use the `Printer`.

Would also be great to go from an `ast.Graph` to a `dot.Graph`. Here I need to evaluate the `ast` as
attributes apply to the current "scope" in order.

Questions
* how to deal represent an `ast.ID`? If I just use a `string` in `dot.Graph.ID` it would lead to an
  invalid ID in the ast. Validate that before? Or deal with such errors later? Or sanitize myself?
