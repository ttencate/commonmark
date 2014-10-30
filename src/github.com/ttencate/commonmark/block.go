package commonmark

import (
	"bytes"
)

// Block represents a node in the parse tree.
//
// "We can think of a document as a sequence of blocks—structural elements like
// paragraphs, block quotations, lists, headers, rules, and code blocks. Blocks
// can contain other blocks, or they can contain inline content: words, spaces,
// links, emphasized text, images, and inline code."
type Block interface {
	// Children returns the list of child blocks, in order.
	Children() []Block

	// AppendChild appends a child block to the list of children.
	AppendChild(Block)

	// AppendLine appends the given line to the list of lines.
	AppendLine([]byte)

	// AcceptsLines returns whether blocks of this type can contain lines.
	AcceptsLines() bool
}

// block implements the common part of the Block interface.
type block struct {
	children      []Block
	content       []byte
	inlineContent Inline
}

func (b *block) Children() []Block {
	return b.children
}

func (b *block) AppendChild(child Block) {
	b.children = append(b.children, child)
}

func (b *block) AppendLine(line []byte) {
	b.content = append(b.content, line...)
}

// document is the root node of the parse tree.
type document struct {
	block
}

func (d *document) AcceptsLines() bool {
	return false
}

// paragraph represents a paragraph: a sequence of non-blank lines that cannot
// be interpreted as other kinds of blocks.
type paragraph struct {
	block
}

func (p *paragraph) AppendLine(line []byte) {
	p.block.AppendLine(bytes.TrimLeft(line, " "))
}

func (p *paragraph) AcceptsLines() bool {
	return true
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
			switch openBlock.(type) {
			case *paragraph:
				break
			}
		}

		if openBlock.AcceptsLines() {
			openBlock.AppendLine(line)
		} else {
			par := &paragraph{}
			par.AppendLine(line)
			openBlock.AppendChild(par)
			openBlocks = append(openBlocks, par)
		}
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

	for _, child := range b.Children() {
		processInlines(child)
	}
}
