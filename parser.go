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

func NewParser(r io.Reader) (*Parser, error) {
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

	return stmts, nil
}

func (p *Parser) parseHeader() (ast.Graph, error) {
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
		graph.ID = &ast.ID{
			Literal:  p.curToken.Literal,
			StartPos: p.curToken.Start,
			EndPos:   p.curToken.End,
		}
	}

	return graph, nil
}

func (p *Parser) parseStatement(graph ast.Graph) (ast.Stmt, error) {
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
				return &ast.NodeStmt{NodeID: nid, AttrList: attrs}, nil
			}

			left = nid
			stmt = &ast.NodeStmt{NodeID: nid}
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
	} else if p.curTokenIs(token.Comment) {
		return p.parseComment()
	} else if p.curTokenIs(token.Equal) {
		return nil, errors.New(`expected an "identifier" before the '='`)
	}

	return nil, nil
}

func (p *Parser) parseEdgeOperand(graph ast.Graph) (ast.EdgeOperand, error) {
	if p.curTokenIs(token.Identifier) {
		return p.parseNodeID()
	}
	subgraph, err := p.parseSubgraph(graph)
	return subgraph, err
}

func (p *Parser) parseEdgeRHS(graph ast.Graph) (ast.EdgeRHS, error) {
	var first, cur *ast.EdgeRHS
	for p.curTokenIsOneOf(token.UndirectedEgde, token.DirectedEgde) {
		operatorStart := p.curToken.Start
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
			first = &ast.EdgeRHS{
				Directed: directed,
				Right:    right,
				StartPos: operatorStart,
			}
			cur = first
		} else {
			cur.Next = &ast.EdgeRHS{
				Directed: directed,
				Right:    right,
				StartPos: operatorStart,
			}
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
	nid := ast.NodeID{
		ID: ast.ID{
			Literal:  p.curToken.Literal,
			StartPos: p.curToken.Start,
			EndPos:   p.curToken.End,
		},
	}

	hasID, err := p.advanceIfPeekTokenIsOneOf(token.Colon)
	if err != nil || !hasID {
		return nid, err
	}

	port, err := p.parsePort()
	if err != nil {
		return nid, err
	}
	nid.Port = port

	return nid, nil
}

func (p *Parser) parsePort() (*ast.Port, error) {
	err := p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return nil, err
	}

	if !p.peekTokenIsOneOf(token.Colon) { // port is either :ID | :compass_pt
		cp, ok := ast.IsCompassPoint(p.curToken.Literal)
		if ok {
			return &ast.Port{
				CompassPoint: &ast.CompassPoint{
					Type:     cp,
					StartPos: p.curToken.Start,
					EndPos:   p.curToken.End,
				},
			}, nil
		}
		return &ast.Port{
			Name: &ast.ID{
				Literal:  p.curToken.Literal,
				StartPos: p.curToken.Start,
				EndPos:   p.curToken.End,
			},
		}, nil
	}

	// port with name and compass_pt :ID:compass_pt
	port := ast.Port{
		Name: &ast.ID{
			Literal:  p.curToken.Literal,
			StartPos: p.curToken.Start,
			EndPos:   p.curToken.End,
		},
	}

	err = p.expectPeekTokenIsOneOf(token.Colon)
	if err != nil {
		return &port, err
	}
	err = p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return &port, err
	}

	cp, ok := ast.IsCompassPoint(p.curToken.Literal)
	if !ok {
		return &port, fmt.Errorf(
			"expected a compass point %v instead got %q",
			[]string{
				ast.CompassPointUnderscore.String(),
				ast.CompassPointNorth.String(),
				ast.CompassPointNorthEast.String(),
				ast.CompassPointEast.String(),
				ast.CompassPointSouthEast.String(),
				ast.CompassPointSouth.String(),
				ast.CompassPointSouthWest.String(),
				ast.CompassPointWest.String(),
				ast.CompassPointNorthWest.String(),
				ast.CompassPointCenter.String(),
			},
			p.curToken.Literal,
		)
	}
	port.CompassPoint = &ast.CompassPoint{
		Type:     cp,
		StartPos: p.curToken.Start,
		EndPos:   p.curToken.End,
	}

	return &port, nil
}

func (p *Parser) parseAttrStatement() (*ast.AttrStmt, error) {
	ns := &ast.AttrStmt{ID: ast.ID{
		Literal:  p.curToken.Literal,
		StartPos: p.curToken.Start,
		EndPos:   p.curToken.End,
	}}

	err := p.expectPeekTokenIsOneOf(token.LeftBracket)
	if err != nil {
		return ns, err
	}

	attrs, err := p.parseAttrList()
	if err != nil {
		return ns, err
	}

	ns.AttrList = *attrs

	return ns, nil
}

func (p *Parser) parseAttrList() (*ast.AttrList, error) {
	var first, cur *ast.AttrList
	for p.curTokenIs(token.LeftBracket) {
		openingBracketStart := p.curToken.Start
		err := p.expectPeekTokenIsOneOf(token.RightBracket, token.Identifier)
		if err != nil {
			return first, err
		}

		// a_list is optional
		var alist *ast.AList
		if p.curTokenIs(token.Identifier) {
			alist, err = p.parseAList()
			if err != nil {
				return first, err
			}
			err = p.expectPeekTokenIsOneOf(token.RightBracket)
			if err != nil {
				return first, err
			}
		}

		if first == nil {
			first = &ast.AttrList{
				AList:    alist,
				StartPos: openingBracketStart,
				EndPos:   p.curToken.End,
			}
			cur = first
		} else {
			cur.Next = &ast.AttrList{
				AList:    alist,
				StartPos: openingBracketStart,
				EndPos:   p.curToken.End,
			}
			cur = cur.Next
		}

		_, err = p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
		if err != nil {
			return first, err
		}
	}

	return first, nil
}

func (p *Parser) parseAList() (*ast.AList, error) {
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
	attr := ast.Attribute{
		Name: ast.ID{
			Literal:  p.curToken.Literal,
			StartPos: p.curToken.Start,
			EndPos:   p.curToken.End,
		},
	}

	err := p.expectPeekTokenIsOneOf(token.Equal)
	if err != nil {
		return attr, err
	}

	err = p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return attr, err
	}
	attr.Value = ast.ID{
		Literal:  p.curToken.Literal,
		StartPos: p.curToken.Start,
		EndPos:   p.curToken.End,
	}

	return attr, nil
}

func (p *Parser) parseSubgraph(graph ast.Graph) (ast.Subgraph, error) {
	subraph := ast.Subgraph{
		StartPos: p.curToken.Start,
	}

	if p.curTokenIs(token.Subgraph) {
		subraph.HasKeyword = true

		// subgraph ID is optional
		hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
		if err != nil {
			return subraph, err
		}

		if hasID {
			subraph.ID = &ast.ID{
				Literal:  p.curToken.Literal,
				StartPos: p.curToken.Start,
				EndPos:   p.curToken.End,
			}
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
	subraph.EndPos = p.curToken.End

	return subraph, nil
}

func (p *Parser) parseComment() (ast.Comment, error) {
	return ast.Comment{
		Text:     string(p.curToken.Literal),
		StartPos: p.curToken.Start,
		EndPos:   p.curToken.End,
	}, nil
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

// expectPeekTokenIsOneOf advances the parser to the peek token if it is one of the wanted tokens.
// Otherwise, the parser position is not changed and an error is returned.
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
