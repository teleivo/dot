# DOT

Parser and formatter for the [DOT language](https://graphviz.org/doc/info/lang.html) written in Go.

## Install

```sh
go get -u github.com/teleivo/dot
```

## Formatter

Format your DOT files with `dotfmt`. `dotfmt` is inspired by [gofmt](https://pkg.go.dev/cmd/gofmt).
As such it is opinionated and has no options to change its format.

Core principles:

* **Idempotency**: Formatting the same code multiple times produces identical output.
* **Only formats valid code**: Parse errors leave the original input unchanged. The formatter does
not assume or alter user intent when it cannot parse the code.

```sh
go run ./cmd/dotfmt/main.go <<EOF
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

analytics[label="Analytics\nDashboard",shape=tab,fillcolor="#fff9c4",color="#f57f17",style="filled,bold"]
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

## Testing

The formatter uses two test strategies:

* Idempotency tests verify formatting is stable
* Visual tests ensure formatting preserves graph semantics by comparing `dot -Tplain` outputs

Run visual tests on external graphs:

```sh
# Sync samples from graphviz repository
./sync-graphviz-samples.sh

cd cmd/dotfmt
DOTFMT_TEST_DIR=../../samples-graphviz/tests go test -v -run TestVisualOutput
```

## Documentation

View the package documentation locally with an interactive example playground:

```sh
# Install pkgsite (Go's documentation server)
go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Run the documentation server
pkgsite -open .
```

This opens a browser with pkg.go.dev-style documentation where you can:

* Read the full package documentation
* View and run the interactive example
* Modify the example code (e.g., change `NewDoc(40)` to different column widths)
* See how the output changes based on your modifications

## Limitations

* does not yet support comments
* does not support https://graphviz.org/doc/info/lang.html#html-strings as I have not needed them
for my purposes
* does not support [double-quoted strings can be concatenated using a '+'
operator](https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting)
* does not treat records in any special way. Labels will be parsed as strings.
* attributes are not validated. For example the color `color="0.650 0.700 0.700"` value has to
* add test for nested subgraphs adhere to some requirements which are not validated. The values are
parsed as identifiers (unquoted, numeral, quoted) and ultimately stored as strings.

## Acknowledgments

The `layout` package is a Go port of [allman](https://github.com/mcy/strings/tree/main/allman) by
mcyoung. The layout algorithm and design are based on the excellent article ["The Art of
Formatting Code"](https://mcyoung.xyz/2025/03/11/formatters/).

## Disclaimer

I wrote this library for my personal projects. It is thus tailored to my needs. Feel free to use it!
That being said, my intention is not to adjust it to someone elses liking.

