package commonmark

import (
	"bytes"
)

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
	content       []byte
	inlineContent Inline
}

func (l *simpleLeaf) AppendLine(line []byte) {
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

func (p *paragraph) AppendLine(line []byte) {
	p.simpleLeaf.AppendLine(bytes.TrimLeft(line, " "))
}

func parseBlocks(data []byte) (*document, error) {
	scanner := newScanner(data)
	doc := &document{}
	openBlocks := []Block{doc}
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')

		var openBlock Block
		for _, openBlock = range openBlocks {
			if _, ok := openBlock.(LeafBlock); ok {
				break
			}
		}

		leafBlock, ok := openBlock.(LeafBlock)
		if !ok {
			containerBlock := openBlock.(ContainerBlock)
			leafBlock = &paragraph{}
			containerBlock.AppendChild(leafBlock)
			openBlocks = append(openBlocks, leafBlock)
		}
		leafBlock.AppendLine(line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

func processInlines(b Block) {
	switch t := b.(type) {
	case *paragraph:
		// "Final spaces are stripped before inline parsing, so a paragraph that
		// ends with two or more spaces will not end with a hard line break."
		t.inlineContent = parseInlines(bytes.TrimRight(t.content, " "))
	}

	if container, ok := b.(ContainerBlock); ok {
		for _, child := range container.Children() {
			processInlines(child)
		}
	}
}
