# DOT

Parser and formatter for the [DOT language](https://graphviz.org/doc/info/lang.html) written in Go.

## Install

```sh
go get -u github.com/teleivo/dot
```

## Formatter

Format your DOT files with `dotfmt`. `dotfmt` is inspired by [gofmt](https://pkg.go.dev/cmd/gofmt).
As such it is opinionated and has no options to change its format.

```sh
go run ./cmd/dotfmt/main.go <<EOF
digraph microservices {
graph [rankdir=LR, bgcolor="#f0f0f0"]
    node [shape=box, style="rounded,filled", fillcolor=lightblue]

    subgraph cluster_frontend {
        label="Frontend Layer"
        style=filled

        web [ label="Web UI",
fillcolor="#8dd3c7"]
        mobile [label="Mobile App", fillcolor="#8dd3c7"]
    }

    api [
  label="API Gateway", shape=hexagon, fillcolor="#ffffb3"
]
    user [label="User Service", fillcolor="#fb8072"]
    db [label="Database", shape=cylinder, fillcolor="#fdb462"]

    web -> api [label="HTTPS", style=bold]
    mobile -> api [label="HTTPS", style=bold]
    api -> user [label="get profile"]
    user -> db [label="read/write"]
}
EOF
```

## Documentation

View the package documentation locally with an interactive example playground:

```sh
# Install pkgsite (Go's documentation server)
go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Run the documentation server
cd layout
pkgsite -open .
```

This opens a browser with pkg.go.dev-style documentation where you can:

* Read the full package documentation
* View and run the interactive example
* Modify the example code (e.g., change `NewDoc(40)` to different column widths)
* See how the output changes based on your modifications

## Limitations

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

