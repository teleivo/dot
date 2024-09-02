// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"errors"
	"fmt"
	"io"
	"slices"

	dot "github.com/teleivo/dot/internal"
	"github.com/teleivo/dot/internal/ast"
	"github.com/teleivo/dot/internal/token"
)

type Parser struct {
	lexer     *dot.Lexer
	curToken  token.Token
	peekToken token.Token
}

func New(r io.Reader) (*Parser, error) {
	lexer, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	p := Parser{
		lexer: lexer,
	}

	// initialize peek token
	err = p.nextToken()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *Parser) nextToken() error {
	p.curToken = p.peekToken
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peekToken = tok
	fmt.Printf("%#v\n", p)

	return nil
}

func (p *Parser) Parse() (ast.Graph, error) {
	// if p.isDone() {
	if p.peekTokenIs(token.EOF) {
		var graph ast.Graph
		return graph, nil
	}

	graph, err := p.parseHeader()
	if err != nil {
		return graph, err
	}
	fmt.Println("after parseHeader")

	err = p.expectPeekTokenIsOneOf(token.LeftBrace)
	if err != nil {
		return graph, err
	}
	// TODO improve/test what if brace is unbalanced/EOF
	err = p.nextToken()
	if err != nil {
		return graph, err
	}

	stmts, err := p.parseStatementList(graph)
	if err != nil {
		return graph, err
	}
	graph.Stmts = stmts

	return graph, err
}

func (p *Parser) parseStatementList(graph ast.Graph) ([]ast.Stmt, error) {
	fmt.Println("parseStatementList")
	var stmts []ast.Stmt
	var err error
	for ; !p.curTokenIsOneOf(token.EOF, token.RightBrace) && err == nil; err = p.nextToken() {
		var stmt ast.Stmt
		stmt, err = p.parseStatement(graph)
		if err != nil {
			return stmts, err
		}

		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	fmt.Println("parseStatementList return")
	return stmts, nil
}

func (p *Parser) parseHeader() (ast.Graph, error) {
	fmt.Println("parseHeader")
	var graph ast.Graph

	err := p.expectPeekTokenIsOneOf(token.Strict, token.Graph, token.Digraph)
	if err != nil {
		return graph, err
	}

	if p.curTokenIs(token.Strict) {
		graph.Strict = true

		err := p.expectPeekTokenIsOneOf(token.Graph, token.Digraph)
		if err != nil {
			return graph, err
		}
	}

	if p.curTokenIs(token.Digraph) {
		graph.Directed = true
	}

	// graph ID is optional
	hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return graph, err
	}

	if hasID {
		graph.ID = p.curToken.Literal
	}

	return graph, nil
}

func (p *Parser) parseStatement(graph ast.Graph) (ast.Stmt, error) {
	fmt.Println("parseStatement")
	if p.curTokenIs(token.Identifier) && p.peekTokenIs(token.Equal) {
		return p.parseAttribute()
	} else if p.curTokenIsOneOf(token.Identifier, token.Subgraph, token.LeftBrace) {
		var stmt ast.Stmt
		var err error

		var left ast.EdgeOperand
		if p.curTokenIs(token.Identifier) {
			nid, err := p.parseNodeID()
			if err != nil {
				return stmt, err
			}

			// attr_list is optional in a node_stmt
			hasLeftBracket, err := p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
			if err != nil {
				return stmt, err
			}
			if hasLeftBracket {
				attrs, err := p.parseAttrList()
				if err != nil {
					return stmt, err
				}
				return &ast.NodeStmt{ID: nid, AttrList: attrs}, nil
			}

			left = nid
			stmt = &ast.NodeStmt{ID: nid}
		} else if p.curTokenIs(token.Subgraph) || p.curTokenIs(token.LeftBrace) {
			subraph, err := p.parseSubgraph(graph)
			if err != nil {
				return stmt, err
			}

			left = subraph
			stmt = subraph
		}

		hasEdgeOperator, err := p.advanceIfPeekTokenIsOneOf(token.UndirectedEgde, token.DirectedEgde)
		if err != nil {
			return stmt, err
		}

		if !hasEdgeOperator {
			return stmt, nil
		}

		es := &ast.EdgeStmt{Left: left}
		erhs, err := p.parseEdgeRHS(graph)
		if err != nil {
			return stmt, err
		}
		es.Right = erhs

		// attr_list is optional in edge_stmt
		hasLeftBracket, err := p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
		if err != nil {
			return es, err
		}
		if !hasLeftBracket {
			return es, nil
		}

		attrs, err := p.parseAttrList()
		if err != nil {
			return es, err
		}

		es.AttrList = attrs

		return es, nil
	} else if p.curTokenIsOneOf(token.Graph, token.Node, token.Edge) {
		return p.parseAttrStatement()
	} else if p.curTokenIs(token.Equal) {
		return nil, errors.New(`expected an "identifier" before the '='`)
	}

	return nil, nil
}

func (p *Parser) parseEdgeOperand(graph ast.Graph) (ast.EdgeOperand, error) {
	fmt.Println("parseEdgeOperand")
	if p.curTokenIs(token.Identifier) {
		return ast.NodeID{ID: p.curToken.Literal}, nil
	}
	subgraph, err := p.parseSubgraph(graph)
	if err != nil {
		return subgraph, err
	}
	return subgraph, nil
}

func (p *Parser) parseEdgeRHS(graph ast.Graph) (ast.EdgeRHS, error) {
	fmt.Println("parseEdgeRHS")
	var first, cur *ast.EdgeRHS
	for p.curTokenIsOneOf(token.UndirectedEgde, token.DirectedEgde) {
		var directed bool
		if p.curTokenIs(token.DirectedEgde) {
			directed = true
		}
		if directed && !graph.Directed {
			return ast.EdgeRHS{}, errors.New("undirected graph cannot contain directed edges")
		}
		if !directed && graph.Directed {
			return ast.EdgeRHS{}, errors.New("directed graph cannot contain undirected edges")
		}

		err := p.expectPeekTokenIsOneOf(token.Identifier, token.Subgraph, token.LeftBrace)
		if err != nil {
			return ast.EdgeRHS{}, err
		}

		right, err := p.parseEdgeOperand(graph)
		if err != nil {
			return ast.EdgeRHS{}, err
		}
		if first == nil {
			first = &ast.EdgeRHS{Directed: directed, Right: right}
			cur = first
		} else {
			cur.Next = &ast.EdgeRHS{Directed: directed, Right: right}
			cur = cur.Next
		}

		hasEdgeOperator, err := p.advanceIfPeekTokenIsOneOf(token.UndirectedEgde, token.DirectedEgde)
		if err != nil {
			return *first, err
		}
		if !hasEdgeOperator {
			return *first, err
		}
	}

	return *first, nil

}

func (p *Parser) parseNodeID() (ast.NodeID, error) {
	fmt.Println("parseNodeID")
	nid := ast.NodeID{ID: p.curToken.Literal}

	hasID, err := p.advanceIfPeekTokenIsOneOf(token.Colon)
	if err != nil || !hasID {
		return nid, err
	}
	err = p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return nid, err
	}
	nid.Port = &ast.Port{ID: p.curToken.Literal}

	return nid, nil
}

func (p *Parser) parseAttrStatement() (*ast.AttrStmt, error) {
	fmt.Println("parseAttrStatement")
	ns := &ast.AttrStmt{ID: p.curToken.Literal}

	err := p.expectPeekTokenIsOneOf(token.LeftBracket)
	if err != nil {
		return ns, err
	}

	attrs, err := p.parseAttrList()
	if err != nil {
		return ns, err
	}

	ns.AttrList = attrs

	return ns, nil
}

func (p *Parser) parseAttrList() (*ast.AttrList, error) {
	fmt.Println("parseAttrList")
	var first, cur *ast.AttrList
	for p.curTokenIs(token.LeftBracket) {
		err := p.expectPeekTokenIsOneOf(token.RightBracket, token.Identifier)
		if err != nil {
			return first, err
		}

		// a_list is optional
		if p.curTokenIs(token.Identifier) {
			alist, err := p.parseAList()
			if err != nil {
				return first, err
			}
			if first == nil {
				first = &ast.AttrList{AList: alist}
				cur = first
			} else {
				cur.Next = &ast.AttrList{AList: alist}
				cur = cur.Next
			}

			err = p.expectPeekTokenIsOneOf(token.RightBracket)
			if err != nil {
				return first, err
			}
		}

		_, err = p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
		if err != nil {
			return first, err
		}
	}

	return first, nil
}

func (p *Parser) parseAList() (*ast.AList, error) {
	fmt.Println("parseAList")
	var first, cur *ast.AList
	for p.curTokenIs(token.Identifier) {
		attr, err := p.parseAttribute()
		if err != nil {
			return first, err
		}
		if first == nil {
			first = &ast.AList{Attribute: attr}
			cur = first
		} else {
			cur.Next = &ast.AList{Attribute: attr}
			cur = cur.Next
		}

		_, err = p.advanceIfPeekTokenIsOneOf(token.Comma, token.Semicolon)
		if err != nil {
			return first, err
		}

		hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
		if err != nil {
			return first, err
		}
		if !hasID {
			return first, err
		}
	}

	return first, nil
}

func (p *Parser) parseAttribute() (ast.Attribute, error) {
	fmt.Println("parseAttribute")
	attr := ast.Attribute{
		Name: p.curToken.Literal,
	}

	err := p.expectPeekTokenIsOneOf(token.Equal)
	if err != nil {
		return attr, err
	}

	err = p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return attr, err
	}
	attr.Value = p.curToken.Literal

	return attr, nil
}

func (p *Parser) parseSubgraph(graph ast.Graph) (ast.Subgraph, error) {
	fmt.Println("parseSubgraph")
	var subraph ast.Subgraph
	if p.curTokenIs(token.Subgraph) {
		// subgraph ID is optional
		hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
		if err != nil {
			return subraph, err
		}

		if hasID {
			subraph.ID = p.curToken.Literal
		}

		err = p.expectPeekTokenIsOneOf(token.LeftBrace)
		if err != nil {
			return subraph, err
		}
	}
	err := p.nextToken()
	if err != nil {
		return subraph, err
	}

	stmts, err := p.parseStatementList(graph)
	if err != nil {
		return subraph, nil
	}
	subraph.Stmts = stmts

	return subraph, nil
}

func (p *Parser) isDone() bool {
	return p.isEOF()
}

func (p *Parser) isEOF() bool {
	return p.curTokenIs(token.EOF)
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) curTokenIsOneOf(tokens ...token.TokenType) bool {
	return slices.ContainsFunc(tokens, p.curTokenIs)
}

func (p *Parser) peekTokenIsOneOf(tokens ...token.TokenType) bool {
	return slices.ContainsFunc(tokens, p.peekTokenIs)
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeekTokenIsOneOf(want ...token.TokenType) error {
	if !p.peekTokenIsOneOf(want...) {
		if len(want) == 1 {
			return fmt.Errorf("expected next token to be %q but got %q instead", want[0], p.peekToken)
		}
		return fmt.Errorf("expected next token to be one of %q but got %q instead", want, p.peekToken)
	}

	err := p.nextToken()
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) advanceIfPeekTokenIsOneOf(tokens ...token.TokenType) (bool, error) {
	if !p.peekTokenIsOneOf(tokens...) {
		return false, nil
	}

	err := p.nextToken()
	if err != nil {
		return true, err
	}

	return true, nil
}
