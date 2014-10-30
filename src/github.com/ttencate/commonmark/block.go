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

	// AcceptsLiteralLines returns whether blocks of this type accept literal
	// lines, which aren't further scanned for subblocks.
	AcceptsLiteralLines() bool

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

func (b *block) AcceptsLiteralLines() bool {
	return false
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

func (c *indentedCodeBlock) AcceptsLiteralLines() bool {
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

// blockQuote represents a block quote; roughly, a series of lines starting
// with '>'.
type blockQuote struct {
	block
}

func (q *blockQuote) AcceptsLines() bool {
	return false
}

func (q *blockQuote) CanContain(Block) bool {
	return true
}

// parseBlocks performs the first parsing pass: turning the document into a
// tree of blocks. Inline content is not parsed at this time.
func parseBlocks(data []byte) (*document, error) {
	doc := &document{}
	parser := blockParser{
		doc:        doc,
		openBlocks: []Block{doc},
	}
	if err := parser.parse(data); err != nil {
		return nil, err
	}
	return doc, nil
}

type blockParser struct {
	doc        *document
	openBlocks []Block
}

func (p *blockParser) addChild(child Block) {
	for i := len(p.openBlocks) - 1; i >= 0; i-- {
		if p.openBlocks[i].CanContain(child) {
			p.openBlocks[i].AppendChild(child)
			p.openBlocks = append(p.openBlocks, child)
			return
		} else {
			p.closeLastBlock()
		}
	}
}

func (p *blockParser) closeLastBlock() {
	p.openBlocks = p.openBlocks[:len(p.openBlocks)-1]
}

func (p *blockParser) openBlock() Block {
	return p.openBlocks[len(p.openBlocks)-1]
}

func (p *blockParser) parse(data []byte) error {
	// See:
	// http://spec.commonmark.org/0.7/#how-source-lines-alter-the-document-tree
	scanner := newScanner(data)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')

		// "The line is analyzed and, depending on its contents, the document
		// may be altered in one or more of the following ways:"

		// "1. One or more open blocks may be closed."
		var openBlock Block
		var i int
		for i, openBlock = range p.openBlocks {
			indent := indentation(line)
			blank := line[indent] == '\n'

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
			case *blockQuote:
				// "3. Consecutiveness. A document cannot contain two block
				// quotes in a row unless there is a blank line between them."
				if blank {
					allMatched = false
				} else if line[indent] == '>' {
					line = stripBlockQuoteMarker(line)
				}
			}
			if !allMatched {
				assertf(i > 0, "allMatched should not become false at the document root")
				i--
				break
			}
		}

		for len(p.openBlocks) > i+1 {
			p.closeLastBlock()
		}

		// "2. One or more new blocks may be created as children of the last open block."
		for !p.openBlock().AcceptsLiteralLines() {
			openBlock := p.openBlock()
			if _, ok := openBlock.(*paragraph); !ok && indentation(line) >= 4 {
				p.addChild(&indentedCodeBlock{})
				line = line[4:]
			} else if line[indentation(line)] == '>' {
				p.addChild(&blockQuote{})
				line = stripBlockQuoteMarker(line)
			} else if isHorizontalRule(line) {
				p.addChild(&horizontalRule{})
				p.closeLastBlock()
				line = nil
				break
			} else if isBlank(line) {
				line = nil
				break
			} else if !openBlock.AcceptsLines() {
				p.addChild(&paragraph{})
			} else {
				break
			}

			if p.openBlock().AcceptsLines() {
				break
			}
		}

		if line == nil {
			continue
		}

		// "3. Text may be added to the last (deepest) open block remaining on
		// the tree."
		openBlock = p.openBlock()
		assertf(openBlock.AcceptsLines(), "remaining types of block should all accept lines, but %T does not (line: %q)", openBlock, line)
		openBlock.AppendLine(line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
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

// isBlank returns whether the line contains only spaces.
func isBlank(line []byte) bool {
	for i, c := range line {
		if c != ' ' {
			return i == len(line)-1
		}
	}
	return true
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

// stripBlockQuoteMarker removes any leading whitespace, the '>' character, and
// optionally a space following that. It assumes that all of this is present.
func stripBlockQuoteMarker(line []byte) []byte {
	line = line[indentation(line)+1:]
	if line[0] == ' ' {
		line = line[1:]
	}
	return line
}

func assertf(condition bool, format string, args ...interface{}) {
	if !condition {
		log.Panicf(format, args...)
	}
}
