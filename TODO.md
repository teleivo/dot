* write cmd/dotfmt
  * hook into conform.nvim
  * try formatting invalid dot and improve error handling
  * what style do I want? should this be documented?

* indentation: which one?
* no ';' unless necessary
* comments should have one " " after the marker
* attribute values:
  * ", " separators instead of just whitespace
  * whitespace separating '=' left from rhs? or no
* whitespace after NodeID if followed by a '{', '[' or EdgeOp
* strip quotes if not necessary?
* newlines: how to make it ok on other platforms?

improve
* handling of EOF better and move these special tokens up top like Go does
* error handling based on feedback from dotfmt

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

* first only format from stdin to stdout
* test using dot examples from gallery
* test using invalid input
* connect vim/conform with dotfmt so gq works with it/formats on write
* uses positional args as files and reads from stdin if non given
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
