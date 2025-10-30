## Scanner

* can I simplify scanner.next()?
* add test for single character/double character scanning. what happens? test the error/eof cases
* think about eof todos and the literal/position I return

scanner error handling
  * go through each invalid test case and think about when to return an actual token instead of
  ILLEGAL and if i should continue consuming for example entire quoted string even if it contains
  invalid characters

// Next advances the scanners position by one token and returns it. The scanner will stop trying to
// tokenize more tokens on the first error it encounters. A token of typen [token.EOF] is returned
// once the underlying reader returns [io.EOF] and the peek token has been consumed.

  the parser should not need a change as it stops on first error

  how to adjust the scanner_test.go? []error? so each token in want has its associated error?
  or []error being all errors that were emitted in order but no nils

For example

```dot
graph {  B
A/
}
```

The scanner can now emit A and then errors on `/`. dotfmt should be able to format this and return
the error(s) that `/` might miss another `/` or `*`.


### BUG: Identifier Lost When Illegal Character Encountered Mid-Scan

**Status:** Discovered 2025-10-30 - Needs investigation and decision

When `tokenizeUnquotedString()` encounters an illegal character in the middle of collecting an
identifier, it returns an ILLEGAL token but **discards the characters collected so far**.

#### Examples

Input: `"A\x00\x00B"`

* **Current behavior:**
  1. ILLEGAL (null byte at 1:2) - 'A' is lost!
  2. ILLEGAL (null byte at 1:3)
  3. IDENTIFIER "B" (at 1:4)

* **Expected behavior:**
  1. IDENTIFIER "A" (at 1:1)
  2. ILLEGAL (null byte at 1:2)
  3. ILLEGAL (null byte at 1:3)
  4. IDENTIFIER "B" (at 1:4)

Input: `"_zab\x7fx"`

* **Current behavior:**
  1. ILLEGAL (\x7f at 1:5) - "_zab" is lost!
  2. IDENTIFIER "x" (at 1:6)

* **Expected behavior:**
  1. IDENTIFIER "_zab" (at 1:1-1:4)
  2. ILLEGAL (\x7f at 1:5)
  3. IDENTIFIER "x" (at 1:6)

#### Verification

```bash
echo -n "A\x00\x00B" | go run ./cmd/tokens/main.go
echo -n "_zab\x7fx" | go run ./cmd/tokens/main.go
```

#### Root Cause

In `scanner.go:344-364`, when an illegal character is detected (line 346):

* The function creates an ILLEGAL token (lines 347-360)
* Returns immediately
* The `id` slice contains collected characters but they're never emitted

#### Potential Fix

When illegal character is encountered:

1. If `len(id) > 0`: break from loop and emit the identifier token
2. The illegal character will be handled on the next `Next()` call
3. If `len(id) == 0`: emit ILLEGAL token immediately (current behavior for illegal at start)

#### Test Impact

The test added at `scanner_test.go:795` (ContinuesScanningAfterError) currently expects the
**buggy behavior** and needs to be updated once this is fixed.

### Comparison with official dot tool behavior

#### Numeral followed by letters (e.g., `1ABC`, `123abc`)

* **dot behavior**: Issues warning "syntax ambiguity - badly delimited number" and splits
  into two tokens (e.g., `1ABC` becomes `1` and `ABC`)
* **My scanner**: Returns ILLEGAL token with error "a numeral can optionally lead with a `-`,
  has to have at least one digit before or after a `.` which must only be followed by digits"
* **Decision needed**: Should I match dot's behavior and split, or keep the error? The spec
  says numerals match `[-]?(.[0-9]+ | [0-9]+(.[0-9]*)?)` which doesn't allow letters.

#### Multiple dots in numerals (e.g., `1.2.3`)

* **dot behavior**: Warning "syntax ambiguity - badly delimited number '1.2.'" and splits into
  `1.2` and `.3`
* **My scanner**: Error "a numeral can only have one `.` that is at least preceded or followed
  by digits"
* **Status**: My scanner correctly rejects this per spec

#### Hyphen/minus in middle of identifiers (e.g., `A-B`, `1-2`)

* **dot behavior**: Syntax error near `-` (treats `-` as potential edge operator)
* **My scanner**: For `A-B`, stops at `A` and tries to parse `-B` as numeral, fails. For
  `1-2`, error "a numeral can only be prefixed with a `-`"
* **Status**: Both reject, but for different reasons. My scanner needs better error messages
  here.

#### String concatenation with `+` (e.g., `"A"+"B"`)

* **dot behavior**: Supports this, concatenates to `AB`
* **My scanner**: Not implemented (spec says "double-quoted strings can be concatenated using
  a '+' operator")
* **Status**: Missing feature (see TODO.md line 325)

#### HTML identifiers (e.g., `<html>`)

* **dot behavior**: Accepts as valid identifier
* **My scanner**: Not implemented (known limitation)
* **Status**: Documented limitation

#### Null bytes

* **dot behavior**: Syntax error in both unquoted and quoted strings
* **My scanner**: Error "illegal character NUL" in unquoted, "missing closing quote" in quoted
* **Status**: Both reject, different error messages

### Missing test cases for quoted/unquoted identifiers

* `"A"B` - quoted followed by unquoted (reverse of `C"D"`)
* `""` - empty quoted string (valid identifier in DOT)
* `A"B"C` - unquoted-quoted-unquoted sandwich pattern
* `"A""B""C"` - multiple consecutive quoted identifiers (currently only tests 2 adjacent)
* `"A"_B` or `A_1"B"` - underscore/digit transitions with quotes

Currently, the "Identifiers" test case at scanner_test.go:212 has `C"D""E"` which covers:
* Unquoted followed by quoted (`C"D"`)
* Adjacent quoted identifiers (`"D""E"`)

But missing the reverse direction and longer chains.

  * handle ./research/error-handling/maxlen.md better

* handle it in parser
* handle it in dotfmt

Error cases to think about and here is what Go does

* LETTERS AFTER NUMBERS

```
   (Non-decimal/non-hex letters stop consumption)
   Decimal + letters: "123xyz"
      Token 1: INT        (pos=0, lit="123")
      Token 2: IDENT      (pos=3, lit="xyz")
      Token 3: ;          (pos=6, lit="\n")
      (No errors)
```

I don't like this as this requires the parser to then flag this as invalid and its way more
complicated to do that than needed. That should be a single token with an error. Its neither a valid
number nor a valid identifier.

* test parser error will keep code as is in dotfmt

* read pros/cons of using reader vs taking in []byte into Parser/Formatter

## Next

* use assertions

* improve error handling [Parser](#parser)

* Move cmd/tokens to example/cmd/tokens or example/tokens? Its not really something I would want to
  be used. Its a mere demo/debugging utility

* do I need the Stringer impls in the AST? would be great to get rid of extra code if not needed.
How to debug/trace then? see Gos trace in the parser. `./cmd/tokens/main.go` is of great help. I
want something similar for the parser. Is it best to integrate that into the scanner/parser or nicer
to keep it externally like `cmd/tokens`?

* support comments
  * line comment
  * support word-wrapping
* support splitting IDs using line-continuation?
* measure in original sets broken if text contains newline. this is not correct for raw strings
right? `foo\nfaa` in Go or similar with escaped newlines or so in DOT should not cause a newline.
add a new tag/attribute? rawtext, `<text raw/>` or don't implement that?

* support stanzas ./samples-graphviz/241_0.dot
  * how do I even know of newlines? Right now I don't generate Breaks based on the tokens
  * implement merging multiple Break() using max(n, m)
    * this was my old todo on that: how to treat newlines? right now they are discarded. Maybe I'd like to group/make blocks.
      Allow users to do that. No more than one empty line though. And will that line be completely
      empty or be indented as the surrounding code?
      I need proper token/ast position. for this row and column

* update README with docs on `dotfmt`
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

## Parser

* make a plan on how to implement
https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html

* I think this should lead to a parser error but does not

```dot
graph {
{1;2;--{3;4}}
}
```

* improve error printing, how to print the line/snippet with ^^^ to highlight were the error is
* implement parser.Trace like the Go parser

* ../graphviz/graphs/directed/russian.gv is confusing as it clearly violates
unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit
https://graphviz.org/doc/info/lang.html#ids

dot -Tsvg <../graphviz/graphs/directed/russian.gv > russian.svg

also works so is that language reference outdated?

* Lexical and Semantic Notes https://graphviz.org/doc/info/lang.html
  * should some of these influence the parser/should it err
  * how does strict affect a graph? no cycles? is that something my parser should validate?

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

### API

* is it nicer to work with slices then my choice of linked lists with *Next whenever there was a
recursive definition?
* should I remove the Directed field from EdgeRHS as that is clear from graph.Directed?
* make error messages more user friendly
  * for example when parsing the attr_stmt the attr_list is mandatory, instead of saying expected [
    I could say that

#### Nice to have

* expose the knowledge of quoted, unquoted, numeral, html identifiers?
* how complicated is it to use the bufio.Readers buffer instead of creating intermediate slices for
identifiers? how much would that even matter at the expense of how much code :sweat_smile:

### Language Feature Support

* support concatenating strings?
https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
> In addition, double-quoted strings can be concatenated using a '+' operator.
* lex html string? or at least deal with it gracefully: see ./samples-graphviz/56.dot

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

* deal with ./maxlen.md

### Hints

* "\n\n\n\t  - F" leads to "a numeral must have at least one digit" pointing to the whitespace
following the -. Is that understandable enough?
* add hints to some errors like <- did you mean ->
* non-breaking space between numerals leads to

echo 'graph{ 100 200 }' | dot -Tsvg -O
Warning: syntax ambiguity - badly delimited number '100' in line 1 of <stdin> splits into two tokens

## dotfmt

* bring back block comments
    * add a test for a multi-line comment like A -- B /* foo */; B -- C

* there should be an off by one error in my mind when it comes to printID as runeCount does not
include the separator \n and I decrement the endColumn to account for prevRune '\'. It does look
like its working though. editors do show different counts for columns :joy: which confuse me. I
guess column count can differ in terms of what they mean.

```go
if endColumn > maxColumn { // the word and \ do not fit on the current line
```

* improve breaking up long lines
  * Only the ID individually is considered right now. In this example `]` exceeds the maxCol

```dot
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
```

* make this prettier

```dot
	B [
		style="filled" //  this should stay with style="filled"
	]
```

the Attribute should go on a new line like above but it ends up looking like

```dot
	B [style="filled" // this should stay with style="filled"
	]
```

comments
    * merge adjacent comments?
    * how to align comments when I do break them up? right now they are not indented at all. indent to
    the level of the previous comment?

* do I need to shield against ASTs generated from code?
* implement isValid and Stringer on token.Position like Go does? the EOF token for example does not
  have a valid token.Position. For example when I don't have a closing '}' for a graph it does not
have a valid EndPos
  see Go ast.BlockStmt docs which mention exactly that
  could help with Nodes like `AttrList` which might be empty

  or make the zero-value valid

* support parsing/formatting ranges
    * parser should be ok with comments before a graph. how to support that in terms of the parser
    API? right now it returns an ast.Graph but the leading comment comes before the ast.Graph
    Can I solve this requirement together with parsing of ranges?

```go
    Parse(io.Reader) ast.Node // at least right now there is no node that would fit the above

    Parse(io.Reader) []ast.Stmt // this could work. In most cases this will be a slice of
    // {ast.Graph} or {ast.Comment, ast.Graph} only but this could also work with parsing a
    // range
```

* test parser with invalid ID as ID for port. check the places were convert literals to
ast.ID without parsing the identifier, should I not parse it first?

* try formatting invalid dot and improve error handling
  * `2->4` leads to error
  "2:15: a numeral can only be prefixed with a `-`"
  allow that :) and turn it into `2 -> 4`
  * LexError return the token.Token.Start token.Position? or return the invalid token at some point?

improve
* handling of EOF better and move these special tokens up top like Go does

* count opening braces and brackets and decrement them on closing to validate they match?
or is that to simplistic as there are rules as to when you are allowed/have to close them?

* how to handle error on fmt.Fprint?
* how to handle errors?

* test using invalid input
  * invalid input should be printed as is, it should not delete user input!

* support formatting file/dirs in place
  * allow passing in file via flag and out file via flag while still allowing stdin/stdout
  * goroutines could be fun once its working ;)
  * format all of https://gitlab.com/graphviz/graphviz/-/tree/main/graphs?ref_type=heads
    * profile, anything obvious I could improve?
    * add a benchmark to ensure no regressions

* add ability to capture execution traces using flight recorder?

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

* support multiple graphs in a file like in samples-graphviz/tests/graphs/multi.gv
* support + on IDs
* join adjacent comments? unless there is an empty newline in between them

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

## Ideas

* how can I make the simplest autocomplete mainly for attributes
  * is an LSP overkill? and if a simple LSP would do can it also provide hot reloading?
* write cmd/dothot hot-reloading a file passing it to dot and showing its svg in the browser
* how could I make something like :InspectTree in neovim for my parser?

