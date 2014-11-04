package commonmark

import (
	"bytes"
	"fmt"
	"io"
)

// NodeContent is the interface that must be implemented by all types of parse
// tree node content, be it inline or block content, container block or leaf
// block.
type NodeContent interface{}

// Node is a node in the abstract syntax tree (parse tree). It is similar to an
// HTML DOM node, except there is only one type; differentation is made based
// on the Content().
type Node struct {
	parent     *Node
	prev       *Node
	next       *Node
	firstChild *Node
	lastChild  *Node
	content    NodeContent
}

// NewNode creates a new, unparented Node with the given content.
func NewNode(content NodeContent) *Node {
	return &Node{content: content}
}

// Content returns the content of the node.
func (n *Node) Content() NodeContent {
	return n.content
}

// SetContent sets new content on the node. nil content can be used to indicate
// container-only nodes, without semantic meaning.
func (n *Node) SetContent(c NodeContent) {
	n.content = c
}

// Parent returns the parent of the node, or nil if it doesn't have any.
func (n *Node) Parent() *Node {
	return n.parent
}

// Prev returns the previous sibling of this node, or nil if it's the first
// child or doesn't have a parent.
func (n *Node) Prev() *Node {
	return n.prev
}

// Next return the next sibling of this node, or nil if it's the last child or
// doesn't have a parent.
func (n *Node) Next() *Node {
	return n.next
}

// FirstChild returns the first child node, or nil if it doesn't have any
// children.
func (n *Node) FirstChild() *Node {
	return n.firstChild
}

// LastChild returns the last child node, or nil if it doesn't have any
// children.
func (n *Node) LastChild() *Node {
	return n.lastChild
}

// PrependChild adds the given node as the first child of this node. The given
// node must have no parent.
func (n *Node) PrependChild(child *Node) {
	child.assertHasNoParent()
	child.parent = n
	child.prev = nil
	child.next = n.firstChild
	if n.firstChild != nil {
		n.firstChild.prev = child
	}
	n.firstChild = child
	if n.lastChild == nil {
		n.lastChild = child
	}
}

// AppendChild adds the given node as the last child of this node. The given
// node must have no parent.
func (n *Node) AppendChild(child *Node) {
	child.assertHasNoParent()
	child.parent = n
	child.prev = n.lastChild
	child.next = nil
	if n.lastChild != nil {
		n.lastChild.next = child
	}
	n.lastChild = child
	if n.firstChild == nil {
		n.firstChild = child
	}
}

// InsertBefore inserts the given node as a sibling right before this node. The
// given node must have no parent; the current node must.
func (n *Node) InsertBefore(sibling *Node) {
	n.assertHasParent()
	sibling.assertHasNoParent()
	if n.prev == nil {
		n.parent.firstChild = sibling
	} else {
		n.prev.next = sibling
	}
	sibling.parent = n.parent
	sibling.prev = n.prev
	sibling.next = n
	n.prev = sibling
}

// InsertBefore inserts the given node as a sibling right after this node. The
// given node must have no parent; the current node must.
func (n *Node) InsertAfter(sibling *Node) {
	n.assertHasParent()
	sibling.assertHasNoParent()
	if n.next == nil {
		n.parent.lastChild = sibling
	} else {
		n.next.prev = sibling
	}
	sibling.parent = n.parent
	sibling.prev = n
	sibling.next = n.next
	n.next = sibling
}

// Remove removes this node from its parent. It must have a parent.
func (n *Node) Remove() {
	n.assertHasParent()
	if n.prev == nil {
		n.parent.firstChild = n.next
	} else {
		n.prev.next = n.next
	}
	if n.next == nil {
		n.parent.lastChild = n.prev
	} else {
		n.next.prev = n.prev
	}
	n.parent = nil
	n.prev = nil
	n.next = nil
}

// String converts the node to a printable string, indenting child nodes and
// showing content types. It ends with a newline.
func (n *Node) String() string {
	buf := bytes.Buffer{}
	n.stringRecurse(&buf, 0)
	return buf.String()
}

func (n *Node) stringRecurse(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		io.WriteString(w, "  ")
	}
	fmt.Fprintf(w, "%T @ %p: %s\n", n.Content(), n, n.Content())
	for child := n.FirstChild(); child != nil; child = child.Next() {
		child.stringRecurse(w, indent+1)
	}
}

func (n *Node) assertHasParent() {
	if n.parent == nil {
		panic("child has no parent")
	}
}

func (n *Node) assertHasNoParent() {
	if n.parent != nil {
		panic("child already has a parent")
	}
}
