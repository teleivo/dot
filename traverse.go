package dot

import "github.com/teleivo/dot/token"

// TreeFirst returns the first child tree matching want.
func TreeFirst(tree *Tree, want TreeKind) (*Tree, bool) {
	return TreeFirstWithin(tree, want, len(tree.Children))
}

// TreeFirstWithin returns the first child tree matching want within [0, last]. Comments are
// skipped.
func TreeFirstWithin(tree *Tree, want TreeKind, last int) (*Tree, bool) {
	var pos int
	for _, child := range tree.Children {
		if tc, ok := child.(TokenChild); ok && tc.Kind == token.Comment {
			continue
		}
		if pos > last {
			break
		}
		if tc, ok := child.(TreeChild); ok && tc.Kind&want != 0 {
			return tc.Tree, true
		}
		pos++
	}
	return nil, false
}

// TreeLast returns the last child tree matching want. Comments are skipped.
func TreeLast(tree *Tree, want TreeKind) (*Tree, bool) {
	for i := len(tree.Children) - 1; i >= 0; i-- {
		child := tree.Children[i]
		if tc, ok := child.(TokenChild); ok && tc.Kind == token.Comment {
			continue
		}
		if tc, ok := child.(TreeChild); ok && tc.Kind&want != 0 {
			return tc.Tree, true
		}
	}
	return nil, false
}

// TreeAt returns the child tree at semantic index if it matches want. Comments are skipped.
func TreeAt(tree *Tree, want TreeKind, at int) (*Tree, bool) {
	var pos int
	for _, child := range tree.Children {
		if tc, ok := child.(TokenChild); ok && tc.Kind == token.Comment {
			continue
		}
		if pos == at {
			if tc, ok := child.(TreeChild); ok && tc.Kind&want != 0 {
				return tc.Tree, true
			}
			return nil, false
		}
		pos++
	}
	return nil, false
}

// TokenFirst returns the first child token matching want.
func TokenFirst(tree *Tree, want token.Kind) (token.Token, bool) {
	tok, _, ok := TokenFirstWithin(tree, want, len(tree.Children))
	return tok, ok
}

// TokenFirstWithin returns the first child token matching want within [0, last]. Comments are
// skipped. The returned index is the semantic index.
func TokenFirstWithin(tree *Tree, want token.Kind, last int) (token.Token, int, bool) {
	var pos int
	for _, child := range tree.Children {
		if tc, ok := child.(TokenChild); ok && tc.Kind == token.Comment {
			continue
		}
		if pos > last {
			break
		}
		if tc, ok := child.(TokenChild); ok && tc.Kind&want != 0 {
			return tc.Token, pos, true
		}
		pos++
	}
	var tok token.Token
	return tok, 0, false
}

// TokenAt returns the child token at semantic index if it matches want. Comments are skipped.
func TokenAt(tree *Tree, want token.Kind, at int) (token.Token, bool) {
	var tok token.Token
	var pos int
	for _, child := range tree.Children {
		if tc, ok := child.(TokenChild); ok && tc.Kind == token.Comment {
			continue
		}
		if pos == at {
			if tc, ok := child.(TokenChild); ok && tc.Kind&want != 0 {
				return tc.Token, true
			}
			return tok, false
		}
		pos++
	}
	return tok, false
}

// FirstID returns the token.ID of the first KindID child tree.
func FirstID(tree *Tree) (token.Token, bool) {
	idTree, ok := TreeFirst(tree, KindID)
	if !ok {
		return token.Token{}, false
	}
	return TokenFirst(idTree, token.ID)
}
