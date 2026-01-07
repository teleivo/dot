package dot

import "github.com/teleivo/dot/token"

// TreeFirst returns the first child tree matching want.
func TreeFirst(tree *Tree, want TreeKind) (*Tree, bool) {
	return TreeFirstWithin(tree, want, len(tree.Children))
}

// TreeFirstWithin returns the first child tree matching want within children[0:last] (inclusive).
func TreeFirstWithin(tree *Tree, want TreeKind, last int) (*Tree, bool) {
	for _, child := range tree.Children {
		if last < 0 {
			break
		}

		if tc, ok := child.(TreeChild); ok && tc.Kind&want != 0 {
			return tc.Tree, true
		}
		last--
	}
	return nil, false
}

// TreeLast returns the last child tree matching want.
func TreeLast(tree *Tree, want TreeKind) (*Tree, bool) {
	for i := len(tree.Children) - 1; i >= 0; i-- {
		if tc, ok := tree.Children[i].(TreeChild); ok && tc.Kind&want != 0 {
			return tc.Tree, true
		}
	}
	return nil, false
}

// TreeAt returns the child tree at index if it matches want.
func TreeAt(tree *Tree, want TreeKind, at int) (*Tree, bool) {
	if at >= len(tree.Children) {
		return nil, false
	}

	if tc, ok := tree.Children[at].(TreeChild); ok && tc.Kind&want != 0 {
		return tc.Tree, true
	}
	return nil, false
}

// TokenFirst returns the first child token matching want.
func TokenFirst(tree *Tree, want token.Kind) (token.Token, bool) {
	tok, _, ok := TokenFirstWithin(tree, want, len(tree.Children))
	return tok, ok
}

// TokenFirstWithin returns the first child token matching want within children[0:last] (inclusive).
func TokenFirstWithin(tree *Tree, want token.Kind, last int) (token.Token, int, bool) {
	for i, child := range tree.Children {
		if last < 0 {
			break
		}

		if tc, ok := child.(TokenChild); ok && tc.Kind&want != 0 {
			return tc.Token, i, true
		}
		last--
	}
	var tok token.Token
	return tok, 0, false
}

// TokenAt returns the child token at index if it matches want.
func TokenAt(tree *Tree, want token.Kind, at int) (token.Token, bool) {
	var tok token.Token
	if at >= len(tree.Children) {
		return tok, false
	}

	if tc, ok := tree.Children[at].(TokenChild); ok && tc.Kind&want != 0 {
		return tc.Token, true
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
