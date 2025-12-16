# DOT

Formatter and parser for the [DOT language](https://graphviz.org/doc/info/lang.html) written in Go.

## Formatter

Format your DOT files with `dotx fmt`. `dotx fmt` is inspired by
[gofmt](https://pkg.go.dev/cmd/gofmt). As such it is opinionated and has no options to change its
format.

### Install

```sh
go install github.com/teleivo/dot/cmd/dotx@latest
```

### Usage

```sh
dotx fmt < input.dot > output.dot
```

Or try it directly:

```sh
dotx fmt <<EOF
digraph data_pipeline{graph[rankdir=TB,bgcolor="#fafafa"]
node[shape=box,style="rounded,filled",fontname="Arial",fontsize=11]
edge[fontname="Arial",fontsize=9,arrowsize=0.8]

subgraph cluster_sources{label="Data Sources"
style="filled,rounded"fillcolor="#e3f2fd"color="#1976d2"penwidth=2
raw_logs[label="Raw Logs",shape=note,fillcolor="#bbdefb",color="#1565c0"]
api_data[label="API Data"shape=note,fillcolor="#bbdefb",color="#1565c0"]}

subgraph cluster_processing {label="Processing Layer"
style="filled,rounded"
fillcolor="#f3e5f5"color="#7b1fa2"
penwidth=2
parser [ label="Parser",shape=component,
fillcolor="#ce93d8",color="#6a1b9a"]validate[label="Validator",shape=component,fillcolor="#ce93d8",color="#6a1b9a"]
transform[label="Transformer",shape=component,fillcolor="#ce93d8",color="#6a1b9a"]}

subgraph cluster_storage{
label="Storage"style="filled,rounded"fillcolor="#e8f5e9"
color="#388e3c"penwidth=2
cache[label="Cache",shape=cylinder,fillcolor="#a5d6a7",color="#2e7d32"]
warehouse[label="Data Warehouse",shape=cylinder,fillcolor="#a5d6a7",color="#2e7d32"]}

analytics[label="Analytics Dashboard",shape=tab,fillcolor="#fff9c4",color="#f57f17",style="filled,bold"]
alerts[label="Alert System",shape=octagon,fillcolor="#ffccbc",color="#d84315",style="filled,bold"]

raw_logs->parser[label="ingest",color="#1976d2",penwidth=1.5]
api_data->parser[label="fetch",color="#1976d2",penwidth=1.5]
parser->validate[label="parse",color="#7b1fa2",penwidth=2]validate->transform[label="clean",color="#7b1fa2",penwidth=2]
transform->cache[label="store",color="#388e3c",style=dashed]
transform->warehouse[label="batch write",color="#388e3c",penwidth=2]
cache->analytics[label="query",color="#f57f17"]
warehouse->analytics[label="aggregate",color="#f57f17",penwidth=1.5]
warehouse->alerts[label="monitor",color="#d84315",style=dotted]}
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
* Modify the example code (e.g., change `NewDoc(40)` to different column widths)
* See how the output changes based on your modifications

## Neovim Plugin

The `nvim/` directory contains a Neovim plugin providing `:Dot inspect` for visualizing the CST in a
split window with live updates and cursor tracking.

### Installation (lazy.nvim)

```lua
return {
  'teleivo/dot',
  ft = 'dot',
  opts = {},
}
```

## Limitations

* the parser and formatter do not yet support comments while the scanner does. I plan to at least
support line comments

The following are not supported as I do not need them
* https://graphviz.org/doc/info/lang.html#html-strings
* [double-quoted strings can be concatenated using a '+'
operator](https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting)
* does not treat records in any special way. Labels will be parsed as strings.
* attributes are not validated. For example the color `color="0.650 0.700 0.700"` value has to
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

