# DOT

Parser and formatter for the [DOT language](https://graphviz.org/doc/info/lang.html) written in Go.

## Install

```sh
go get -u github.com/teleivo/dot
```

## Formatter

Format your DOT files with `dotfmt`. `dotfmt` is inspired by [gofmt](https://pkg.go.dev/cmd/gofmt).
As such it is opinionated and has no options to change its format.

TODO complete example
```sh
go run ./cmd/dotfmt/main.go <<EOF
EOF
```

## Limitations

* does not support https://graphviz.org/doc/info/lang.html#html-strings as I have not needed them
for my purposes
* does not support [double-quoted strings can be concatenated using a '+'
operator](https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting)
* does not treat records in any special way. Labels will be parsed as strings.
* attributes are not validated. For example the color `color="0.650 0.700 0.700"` value has to
adhere to some requirements which are not validated. The values are parsed as identifiers (unquoted, numeral, quoted) and ultimately stored as strings.

## Disclaimer

I wrote this library for my personal projects. It is thus tailored to my needs. Feel free to use it!
That being said, my intention is not to adjust it to someone elses liking.

