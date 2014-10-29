package commonmark

// Block represents a node in the parse tree. It can either be a ContainerBlock
// or a LeafBlock.
type Block interface {
}

// ContainerBlock represents a block that can contain other blocks.
type ContainerBlock interface {
	Block

	// Children returns the list of child blocks, in order.
	Children() []Block

	// AppendChild appends a child block to the list of children.
	AppendChild(Block)
}

// LeafBlock represents a block that cannot contain other blocks.
type LeafBlock interface {
	Block

	// AppendLine appends the given line to the list of lines.
	AppendLine([]byte)
}

// simpleContainer implements the ContainerBlock interface in the naive way.
type simpleContainer struct {
	children []Block
}

func (c *simpleContainer) Children() []Block {
	return c.children
}

func (c *simpleContainer) AppendChild(b Block) {
	c.children = append(c.children, b)
}

// simpleLeaf implements the LeafBlock interface in the naive way.
type simpleLeaf struct {
	content []byte
}

func (l *simpleLeaf) AppendLine(line []byte) {
	if len(l.content) > 0 {
		l.content = append(l.content, '\n')
	}
	l.content = append(l.content, line...)
}

// document is the root node of the parse tree.
type document struct {
	simpleContainer
}

// paragraph represents a paragraph: a sequence of non-blank lines that cannot
// be interpreted as other kinds of blocks.
type paragraph struct {
	simpleLeaf
}
