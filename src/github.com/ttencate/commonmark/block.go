package commonmark

import (
	"bytes"
	"log"
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

// paragraph represents a paragraph of text.
//
// "A sequence of non-blank lines that cannot be interpreted as other kinds of
// blocks forms a paragraph."
type paragraph struct {
	block
}

func (p *paragraph) AppendLine(line []byte) {
	p.block.AppendLine(bytes.TrimLeft(line, " "))
}

func (p *paragraph) AcceptsLines() bool {
	return true
}

// indentedCodeBlock represents an indented code block.
//
// "An indented code block is composed of one or more indented chunks separated
// by blank lines. An indented chunk is a sequence of non-blank lines, each
// indented four or more spaces."
type indentedCodeBlock struct {
	block
}

func (c *indentedCodeBlock) AcceptsLines() bool {
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

		indent := indentation(line)
		blank := line[indent] == '\n'

		var openBlock Block
		var i int
		for i, openBlock = range openBlocks {
			allMatched := true
			switch openBlock.(type) {
			case *indentedCodeBlock:
				if indent >= 4 || blank {
					if len(line) > 4 {
						line = line[4:]
					} else {
						line = line[len(line)-1:]
					}
				} else {
					allMatched = false
				}
			case *paragraph:
				// TODO close paragraph on blank line
				break
			}
			if !allMatched {
				assertf(i > 0, "allMatched should not become false at the document root")
				openBlock = openBlocks[i-1]
				break
			}
		}

		if openBlock.AcceptsLines() {
			// We're good.
		} else if indentation(line) >= 4 {
			code := &indentedCodeBlock{}
			openBlock.AppendChild(code)
			openBlocks = append(openBlocks, code)
			openBlock = code
			line = line[4:]
		} else if blank {
			continue
		} else {
			par := &paragraph{}
			openBlock.AppendChild(par)
			openBlocks = append(openBlocks, par)
			openBlock = par
		}
		assertf(openBlock.AcceptsLines(), "remaining types of block should all accept lines, but %T does not", openBlock)
		openBlock.AppendLine(line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

// indentation returns the index of the first non-space. If the line consists
// entirely of spaces, it returns the index of the newline character.
func indentation(line []byte) int {
	for i, c := range line {
		if c != ' ' {
			return i
		}
	}
	assertf(false, "indentation() expects line %q to end in newline character", line)
	return 0
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

func assertf(condition bool, format string, args ...interface{}) {
	if !condition {
		log.Panicf(format, args...)
	}
}
