# DOT Language Grammar Specification

This is the formal grammar for the DOT language used by Graphviz, extracted from the [official
documentation](https://www.graphviz.org/doc/info/lang.html).

## Grammar Production Rules

```
graph      : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'

stmt_list  : [ stmt [ ';' ] stmt_list ]

stmt       : node_stmt
           | edge_stmt
           | attr_stmt
           | ID '=' ID
           | subgraph

attr_stmt  : (graph | node | edge) attr_list

attr_list  : '[' [ a_list ] ']' [ attr_list ]

a_list     : ID '=' ID [ (';' | ',') ] [ a_list ]

edge_stmt  : (node_id | subgraph) edgeRHS [ attr_list ]

edgeRHS    : edgeop (node_id | subgraph) [ edgeRHS ]

node_stmt  : node_id [ attr_list ]

node_id    : ID [ port ]

port       : ':' ID [ ':' compass_pt ]
           | ':' compass_pt

subgraph   : [ subgraph [ ID ] ] '{' stmt_list '}'

compass_pt : (n | ne | e | se | s | sw | w | nw | c | _)
```

## Notes

* Keywords `node`, `edge`, `graph`, `digraph`, `subgraph`, and `strict` are case-independent
* The edge operator (`edgeop`) is:
  * `->` for directed graphs (digraph)
  * `--` for undirected graphs (graph)
* Semicolons are optional statement separators
* Square brackets `[ ]` in the grammar denote optional elements
* Parentheses `( )` group alternatives
* Vertical bar `|` separates alternatives

## ID Formats

Valid identifiers (ID) include:

* Alphanumeric strings and underscores (cannot start with a digit)
* Numeric values (integer or floating-point)
* Quoted strings with escaped quotes (`\"`)
* HTML strings enclosed in angle brackets `<...>`
