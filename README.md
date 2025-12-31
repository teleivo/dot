# DOT

A toolchain for the [DOT language](https://graphviz.org/doc/info/lang.html). Includes `dotx lsp` for
editor integration, `dotx fmt` for formatting, `dotx watch` for live preview and `dotx inspect` for
examining syntax.

## Install

```sh
go install github.com/teleivo/dot/cmd/dotx@latest
```

## LSP

`dotx lsp` starts a Language Server Protocol server for DOT files.

Features:

* Diagnostics (syntax errors as you type)
* Formatting
* Attribute completion with context-aware filtering (node, edge, graph attributes)

## Formatter

Format your DOT files with `dotx fmt`. `dotx fmt` is inspired by
[gofmt](https://pkg.go.dev/cmd/gofmt). As such it is opinionated and has no options to change its
format.

### Usage

```sh
dotx fmt < input.dot > output.dot
```

Or try it directly:

```sh
dotx fmt <<EOF
digraph git{node[shape=rect]"22a1e48"->"abd0f59"->"e83ea81"[label="main"]"22a1e48"->"b4ec655"[label="lsp"]"b4ec655"->"c4a3242"->"e38f243"->"02314ea" "02314ea"->"6504ef3"[label="partial sync"]}
EOF
```

### Design principles

* **No configuration**: `dotx fmt` is opinionated and has no options to change its format.
* **Idempotency**: Formatting the same code multiple times produces identical output.
* **Only formats valid code**: Parse errors are reported to stderr and no output is produced. The
  formatter does not output partial or malformed results.

### Testing

`dotx fmt` uses two test strategies:

* Idempotency tests verify formatting is stable
* Visual tests ensure formatting preserves graph semantics by comparing `dot -Tplain` outputs

Run visual tests on external graphs:

```sh
# Sync samples from the Graphviz repository (https://gitlab.com/graphviz/graphviz)
./sync-graphviz-samples.sh

# Run from repository root
DOTX_TEST_DIR=../../samples-graphviz/tests go test -C cmd/dotx -v -run TestVisualOutput

# For comprehensive testing of all sample directories
./run-visual-tests.sh
```

Note: Some tests will fail due to [known limitations](#limitations) such as HTML labels and
comments. These failures are expected and indicate features not yet supported rather than bugs.

## Inspect

`dotx inspect` provides commands for examining DOT source code structure.

### Tree

Print the concrete syntax tree (CST) representation:

```sh
echo 'digraph { a -> b }' | dotx inspect tree
```

Output:

```
File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'a'
				'->'
				NodeID
					ID
						'b'
		'}'
```

Use `-format=scheme` for a scheme-like representation with positions.

### Tokens

Print the token stream:

```sh
echo 'digraph { a -> b }' | dotx inspect tokens
```

Output:

```
POSITION   TYPE     LITERAL  ERROR
1:1-1:7    digraph  digraph
1:9        {        {
1:11       ID       a
1:13-1:14  ->       ->
1:16       ID       b
1:18       }        }
```

## Watch

Preview DOT files as SVG in your browser with live reload.

Requires the `dot` executable from [Graphviz](https://graphviz.org/download/).

```sh
dotx watch graph.dot
```

The file watcher is designed for editors that use atomic writes (rename temp file to target), such
as Neovim and Vim. Editors that write files in multiple steps may cause brief flashes of errors or
partial content.

## Documentation

View the package documentation locally with an interactive example playground:

```sh
# Install pkgsite (Go's documentation server)
go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Run the documentation server
pkgsite -open .
```

This opens a browser with [pkg.go.dev-style](https://pkg.go.dev) documentation where you can:

* Read the full package documentation
* View and run the interactive example
* Modify the example code (e.g., change `NewDoc(40)` to different column widths to see how the
  layout algorithm reflows text to fit within the specified maximum)

## Neovim

### Plugin

The `nvim/` directory contains a Neovim plugin with commands:

* `:Dot inspect` - visualize the CST in a split window with live updates and cursor tracking
* `:Dot watch` - start `dotx watch` and open the browser for live SVG preview

Installation with lazy.nvim:

```lua
return {
  'teleivo/dot',
  ft = 'dot',
  opts = {},
}
```

### LSP Configuration

Neovim 0.11+ with `lsp/` directory (see `:help lsp-config`):

```lua
-- ~/.config/nvim/lsp/dotls.lua
return {
  cmd = { 'dotx', 'lsp' },
  filetypes = { 'dot' },
}
```

```lua
-- ~/.config/nvim/init.lua
vim.lsp.enable('dotls')
```

## Limitations

* The scanner assumes UTF-8 encoded input. Invalid UTF-8 byte sequences are replaced with the
  Unicode replacement character (U+FFFD) and reported as errors. Files in other encodings (UTF-16,
  Latin-1, etc.) must be converted to UTF-8 first.
* The LSP server only supports UTF-8 position encoding. According to the LSP specification, servers
  must support UTF-16 as the default. However, `dotx lsp` always uses UTF-8 regardless of what the
  client offers. This works correctly with clients that support UTF-8 (such as Neovim) but may cause
  incorrect character positions with clients that only support UTF-16.
* The formatter uses Unicode code points (runes) for measuring text width and line length. This does
  not account for grapheme clusters or display width, so characters like emojis (which may render as
  double-width) or combining characters will cause the formatter's column calculations to differ
  from visual appearance in editors.
* The parser and formatter do not yet support comments while the scanner does. I plan to at least
  support line comments.

The following are not supported as I do not need them:

* https://graphviz.org/doc/info/lang.html#html-strings
* [Double-quoted strings can be concatenated using a '+'
  operator](https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting)
* Does not treat records in any special way. Labels will be parsed as strings.
* Attributes are not validated. For example the color `color="0.650 0.700 0.700"` value has to
  adhere to some requirements which are not validated. The values are parsed as IDs (unquoted,
  numeral, quoted) and ultimately stored as strings.

## Acknowledgments

The parser uses a homogeneous tree structure and practical error recovery techniques inspired by
matklad's [Resilient LL Parsing
Tutorial](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html). The full
event-based two-phase parsing approach was too complex for a simple language like DOT.

The `layout` package is a Go port of [allman](https://github.com/mcy/strings/tree/main/allman) by
mcyoung. The layout algorithm and design are based on the excellent article ["The Art of Formatting
Code"](https://mcyoung.xyz/2025/03/11/formatters/).

## Disclaimer

I wrote this library for my personal projects and it is provided as-is without warranty. It is
tailored to my needs and my intention is not to adjust it to someone else's liking. Feel free to use
it!

See [LICENSE](LICENSE) for full license terms.

