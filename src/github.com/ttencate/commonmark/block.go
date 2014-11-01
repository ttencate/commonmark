package commonmark

import (
	"log"
)

// Document is the NodeContent for the root node.
type Document struct{}

// UnprocessedLine is the NodeContent for lines whose role hasn't yet been
// determined. At the end of the block parsing phase, all of these must be
// gone.
type UnprocessedLine struct {
	// Content is the content of this line, including a terminating '\n'
	// (regardless of input line ending type).
	Content []byte
}

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
}

func assertf(condition bool, format string, args ...interface{}) {
	if !condition {
		log.Panicf(format, args...)
	}
}
