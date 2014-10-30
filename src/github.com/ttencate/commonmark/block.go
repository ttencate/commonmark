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

	// CanContain returns whether this block can contain the given block.
	CanContain(Block) bool
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

func (d *document) CanContain(Block) bool {
	return true
}

// horizontalRule is a horizontal rule.
//
// "A line consisting of 0-3 spaces of indentation, followed by a sequence of
// three or more matching -, _, or * characters, each followed optionally by
// any number of spaces, forms a horizontal rule."
type horizontalRule struct {
	block
}

func (r *horizontalRule) AcceptsLines() bool {
	return false
}

func (r *horizontalRule) CanContain(Block) bool {
	return false
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

func (c *indentedCodeBlock) CanContain(Block) bool {
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

func (p *paragraph) CanContain(Block) bool {
	return false
}

// parseBlocks performs the first parsing pass: turning the document into a
// tree of blocks. Inline content is not parsed at this time. See:
// http://spec.commonmark.org/0.7/#how-source-lines-alter-the-document-tree
func parseBlocks(data []byte) (*document, error) {
	scanner := newScanner(data)
	doc := &document{}
	openBlocks := []Block{doc}
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')

		// "The line is analyzed and, depending on its contents, the document
		// may be altered in one or more of the following ways:"
		indent := indentation(line)
		blank := line[indent] == '\n'

		// "1. One or more open blocks may be closed."
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
				if blank {
					allMatched = false
				}
				break
			}
			if !allMatched {
				assertf(i > 0, "allMatched should not become false at the document root")
				i--
				break
			}
		}

		openBlocks = openBlocks[:i+1]
		openBlock = openBlocks[len(openBlocks)-1]

		// "2. One or more new blocks may be created as children of the last open block."
		if _, ok := openBlock.(*paragraph); !ok && indentation(line) >= 4 {
			code := &indentedCodeBlock{}
			openBlock.AppendChild(code)
			openBlocks = append(openBlocks, code)
			openBlock = code
			line = line[4:]
		} else if _, ok := openBlock.(*paragraph); ok {
			// Fall through.
		} else if isHorizontalRule(line) {
			openBlock.AppendChild(&horizontalRule{})
			continue
		} else if blank {
			continue
		} else if !openBlock.AcceptsLines() {
			par := &paragraph{}
			openBlock.AppendChild(par)
			openBlocks = append(openBlocks, par)
			openBlock = par
		}

		// "3. Text may be added to the last (deepest) open block remaining on
		// the tree."
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

// isHorizontalRule returns whether the line contains a valid horizontal rule.
func isHorizontalRule(line []byte) bool {
	var char byte
	var count int
	for i, c := range line {
		// "... each followed optionally by any number of spaces ..."
		if c != ' ' && c != '\n' {
			if c != '-' && c != '_' && c != '*' {
				return false
			}
			// "... matching -, _, or * characters ..."
			if char == 0 {
				if i >= 4 {
					// "A line consisting of 0-3 spaces of indentation ..."
					return false
				}
				char = c
				count = 1
			} else if c == char {
				count++
			} else {
				return false
			}
		}
	}
	// "... a sequence of three or more ..."
	return count >= 3
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
