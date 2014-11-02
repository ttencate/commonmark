package commonmark

import (
	"log"
)

// Document is the NodeContent for the root node.
type Document struct{}

// RawLine is the NodeContent for lines whose role hasn't yet been
// determined. At the end of the block parsing phase, all of these must be
// gone.
type RawLine struct {
	// Content is the content of this line, including a terminating '\n'
	// (regardless of input line ending type).
	Content []byte

	// Indentation is the number of space (' ') characters at the start of the
	// line.
	Indentation int

	// IsBlank is true if the line consists only of space characters.
	IsBlank bool

	// FirstNonSpaceChar is the first character (byte, really) of the line that
	// is not a space. If the line is blank, this will be '\n'.
	FirstNonSpaceChar byte
}

// Paragraph is the NodeContent for a paragraph of text. Before inline
// processing, it will contain a single Text node.
type Paragraph struct{}

// parseBlocks performs the first parsing pass: turning the document into a
// tree of blocks. Inline content is not parsed at this time.
func parseBlocks(root *Node) {
	parser := blockParser{root: root}
	parser.parse()
}

type blockParser struct {
	root *Node
}

func (p *blockParser) parse() {
	p.processParagraphs(p.root)
}

func (p *blockParser) processParagraphs(n *Node) {
	// "A sequence of non-blank lines that cannot be interpreted as other kinds
	// of blocks forms a paragraph. The contents of the paragraph are the
	// result of parsing the paragraph’s raw content as inlines. The
	// paragraph’s raw content is formed by concatenating the lines and
	// removing initial and final spaces."
	var rawText *RawText
	for child := n.FirstChild(); child != nil; {
		if rawLine, ok := child.Content().(*RawLine); ok {
			if rawText == nil {
				rawText = &RawText{}
				par := NewNode(&Paragraph{})
				par.AppendChild(NewNode(rawText))
				child.InsertBefore(par)
			}
			rawText.Content = append(rawText.Content, rawLine.Content...)
			child = removeAndNext(child)
		} else {
			rawText = nil
			child = child.Next()
		}
	}
}

func removeAndNext(n *Node) *Node {
	next := n.Next()
	n.Remove()
	return next
}

func assertf(condition bool, format string, args ...interface{}) {
	if !condition {
		log.Panicf(format, args...)
	}
}
