package commonmark

import (
	"log"
)

// Document is the NodeContent for the root node.
type Document struct{}

// parseBlocks performs the first parsing pass: turning the document into a
// tree of blocks. Inline content is not parsed at this time.
func parseBlocks(data []byte) (*Node, error) {
	root := NewNode(&Document{})
	parser := blockParser{
		root: root,
	}
	if err := parser.parse(data); err != nil {
		return nil, err
	}
	return root, nil
}

type blockParser struct {
	root *Node
}

func (p *blockParser) parse(data []byte) error {
	scanner := newScanner(data)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func assertf(condition bool, format string, args ...interface{}) {
	if !condition {
		log.Panicf(format, args...)
	}
}
