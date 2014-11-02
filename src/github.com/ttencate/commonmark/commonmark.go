// Package commonmark provides functionality to convert CommonMark syntax to
// HTML.
package commonmark

import (
	"bytes"
)

// ToHTMLBytes converts text formatted in CommonMark into the corresponding
// HTML.
//
// The input must be encoded as UTF-8.
//
// Line breaks in the output will be single '\n' bytes, regardless of line
// endings in the input (which can be CR, LF or CRLF).
//
// Note that the output might contain unsafe tags (e.g. <script>); if you are
// accepting untrusted user input, you must run the output through a sanitizer
// before sending it to a browser.
func ToHTMLBytes(data []byte) ([]byte, error) {
	root, err := parse(data)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	ToHTML(root, &buffer)
	return buffer.Bytes(), nil
}

func parse(data []byte) (*Node, error) {
	root, err := preprocess(data)
	if err != nil {
		return nil, err
	}

	// See http://spec.commonmark.org/0.7/#appendix-a-a-parsing-strategy
	// "Parsing has two phases:"

	// "In the first phase, lines of input are consumed and the block structure
	// of the document—its division into paragraphs, block quotes, list items,
	// and so on—is constructed. Text is assigned to these blocks but not
	// parsed. Link reference definitions are parsed and a map of links is
	// constructed."
	parseBlocks(root)

	// "In the second phase, the raw text contents of paragraphs and headers
	// are parsed into sequences of Markdown inline elements (strings, code
	// spans, links, emphasis, and so on), using the map of link references
	// constructed in phase 1."
	processInlines(root)

	return root, nil
}

func processInlines(n *Node) {
	switch t := n.Content().(type) {
	case *RawText:
		parseInlines(n, t.Content)
	}

	for child := n.FirstChild(); child != nil; child = child.Next() {
		processInlines(child)
	}
}
