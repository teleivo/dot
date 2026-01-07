# TODO

## Jan

week 2
* fmt/lsp: support comments

week 3
* fmt: format files/directories

week 4
* profile fmt/lsp
  * `dotx fmt < samples-graphviz/share/examples/world.gv` is the most challenging
  * add ability to capture execution traces using flight recorder?

week 5
* skeleton:
  * lrb using my dotx toolchain and visual .dot files for test errors and state
  * invariant check

## Parser

* add recursion depth limit to prevent stack overflow on deeply nested subgraphs
* commas: the parser handles commas only in attribute lists (`[a=1, b=2]`) per the official DOT
  grammar. However, Graphviz itself is more permissive and accepts commas as statement/element
  separators elsewhere:
  * `a, b` - comma between statements (Graphviz accepts, grammar forbids)
  * `a -> b, c` - comma in edge RHS (Graphviz accepts, grammar forbids)
  * `{b, c}` - comma in node group (Graphviz accepts, grammar forbids)

  Decision: follow the grammar for now. Consider matching Graphviz's permissive behavior later if
  needed for real-world compatibility.

## fmt

* support comments
  * line comments only at first
  * support word-wrapping
  * how to align comments when breaking them up? right now they are not indented at all
  * add a test for a multi-line comment like `A -- B /* foo */; B -- C`
* measure in original sets broken if text contains newline - not correct for raw strings?
  `foo\nfaa` in Go or similar with escaped newlines in DOT should not cause a newline. Add a new
  tag/attribute? rawtext, `<text raw/>` or don't implement that?
* support stanzas ./samples-graphviz/241_0.dot
  * how do I even know of newlines? Right now I don't generate Breaks based on the tokens
  * implement merging multiple Break() using max(n, m)
  * how to treat newlines? right now they are discarded. Maybe allow users to group/make blocks.
    No more than one empty line though. Need proper token/ast position with row and column.
* lex html string? or at least deal with it gracefully: see ./samples-graphviz/56.dot
* layout uses `len(tag.content)` (bytes) not rune count - may miscount width for non-ASCII
* improve breaking up long lines - only the ID individually is considered right now:

```dot
"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
```

  In this example `]` exceeds the maxCol.
* improve error printing - print the line/snippet with ^^^ to highlight where the error is
  * make error messages more user friendly - for example when parsing attr_stmt the attr_list is
    mandatory, instead of saying "expected [" could say that
* count opening braces and brackets and decrement on closing to validate they match? Or is that
  too simplistic as there are rules as to when you are allowed/have to close them?
* support parsing/formatting ranges
  * parser should be ok with comments before a graph - how to support that in terms of the parser
    API? right now it returns an ast.Graph but the leading comment comes before the ast.Graph
  * Can I solve this requirement together with parsing of ranges?

```go
Parse(io.Reader) ast.Node // at least right now there is no node that would fit the above

Parse(io.Reader) []ast.Stmt // this could work. In most cases this will be a slice of
// {ast.Graph} or {ast.Comment, ast.Graph} only but this could also work with parsing a range
```

* support formatting file/dirs in place
  * allow passing in file via flag and out file via flag while still allowing stdin/stdout
  * goroutines could be fun once it's working
  * format all of https://gitlab.com/graphviz/graphviz/-/tree/main/graphs?ref_type=heads
  * add a benchmark to ensure no regressions
  * gofumpt uses positional args as files and reads from stdin if none given
  * gofumpt hint on formatting pieces of Go: tries `parser.ParseFile`, on error adds `package p;`
    and tries again. If that fails, wraps in package with function and tries ParseFile again.
    Uses `;` so line numbers stay correct. I could try parsing a Graph, if that fails wrap in
    `graph { }` assuming src is []Stmt. Might fail if src contains directed edges so detect and
    try with `digraph {}`.

## LSP

* look into debouncing diagnostics. delay publishing by ~100ms, cancel if another change
arrives. joining } of subgraph onto line above it causes brief flashes of errors as neovim sends an
insert and then a delete as separate changes

```dot
graph foo {
	subgraph cluster_faa {
		1 -- 18
	}
}
```

## CLI

## Performance

* profile and improve performance
  * use unique/string interning?
  * can I make use of this in 1.26? https://github.com/golang/go/issues/73794
  * improve layout printing and reduce overhead of fmt especially for writing '\t' or '\n'
* should I buffer the given w writers in my Render/Print functions?

## Testing

* can I use fuzzing?
* or the https://graphviz.org/docs/cli/gvgen/

## Questions

* ../graphviz/graphs/directed/russian.gv is confusing as it clearly violates "unquoted string
  identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or
  digits([0-9]), but not begin with a digit" https://graphviz.org/doc/info/lang.html#ids
  `dot -Tsvg <../graphviz/graphs/directed/russian.gv > russian.svg` also works - is that language
  reference outdated?
* Lexical and Semantic Notes https://graphviz.org/doc/info/lang.html
  * should some of these influence the parser/should it err?
  * how does strict affect a graph? no cycles? is that something my parser should validate?
* Are `{}` creating a lexical scope? This sets attributes on given nodes in the `{}` but will it
  affect nodes outside?

```dot
{ node [shape=circle]
    a b c d e f g h  i j k l m n o p  q r s t u v w x
}
{ node [shape=diamond]
    A B C D E F G H  I J K L M N O P  Q R S T U V W X
}
```

