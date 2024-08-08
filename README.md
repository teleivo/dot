# DOT

Parser for the [DOT language](https://graphviz.org/doc/info/lang.html) written in Go.

## Install

```sh
go get -u github.com/teleivo/dot
```

**Needs: export GOEXPERIMENT=rangefunc** as it uses the experimental [iterators](https://go.dev/wiki/RangefuncExperiment).

## Limitations

* does not produce an [AST](https://en.wikipedia.org/wiki/Abstract_syntax_tree) but a data structure
  representing the graph, edges and nodes
* comments are discarded as this parser does not produce an AST
* does not support https://graphviz.org/doc/info/lang.html#html-strings as I have not needed them
for my purposes

## Disclaimer

I wrote this library for my personal projects. It is thus tailored to my needs. Feel free to use it!
That being said, my intention is not to adjust it to someone elses liking.

