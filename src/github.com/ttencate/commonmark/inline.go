package commonmark

import (
	"bytes"
)

// Text is the NodeContent for text strings, sent to the output verbatim (but
// possibly with escaping).
type Text struct {
	Content []byte
}

func parseInlines(parent *Node, data []byte) {
	parser := inlineParser{
		parent: parent,
		data:   data,
	}
	parser.parse()
}

type inlineParser struct {
	parent *Node
	data   []byte
}

func (p *inlineParser) parse() {
	// I can't find where the spec decrees this. But the reference
	// implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	// Filed issue:
	// https://github.com/jgm/CommonMark/issues/176
	text := trimWhitespaceRight(p.data)
	p.parent.AppendChild(NewNode(&Text{text}))
}

var asciiPunct = []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

func isASCIIPunct(char byte) bool {
	return bytes.IndexByte(asciiPunct, char) >= 0
}

func trimWhitespaceRight(data []byte) []byte {
	var i int
	for i = len(data) - 1; i > 0; i-- {
		c := data[i-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' && c != '\f' {
			break
		}
	}
	return data[:i]
}

func collapseSpace(data []byte) []byte {
	var out []byte
	var prevWasSpace bool
	for _, c := range data {
		if c == ' ' || c == '\n' {
			if !prevWasSpace {
				out = append(out, ' ')
				prevWasSpace = true
			}
		} else {
			out = append(out, c)
			prevWasSpace = false
		}
	}
	return out
}
