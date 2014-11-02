package commonmark

import (
	"bytes"
	"regexp"
)

// RawText is the NodeContent for text that still needs to go through inline
// parsing. Nodes of this type cannot have children.
type RawText struct {
	Content []byte
}

// Text is the NodeContent for text strings, sent to the output verbatim (but
// possibly with escaping). Nodes of this type cannot have children.
type Text struct {
	Content []byte
}

// HardLineBreak is the NodeContent for hard line breaks, rendered as <br />
// tags in HTML.
type HardLineBreak struct{}

// parseInlines processes inline elements on the given node. It returns a new
// Node reflecting the parsed version of the given text.
func parseInlines(text []byte) *Node {
	// I can't find where the spec decrees this. But the reference
	// implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	// Filed issue:
	// https://github.com/jgm/CommonMark/issues/176
	text = trimWhitespaceRight(text)
	n := NewNode(&Text{text})

	n = processHardLineBreaks(n)

	return n
}

var asciiPunct = []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

// isASCIIPunct mimics the behaviour of ispunct(3). It is neither a subset nor
// a superset of unicode.IsPunct; for instance, '`' is not considered
// punctuation by Unicode.
func isASCIIPunct(char byte) bool {
	// TODO speed this up using a []bool
	return bytes.IndexByte(asciiPunct, char) >= 0
}

// trimWhitespaceRight returns a subslice where all trailing whitespace (ASCII
// only) is removed.
func trimWhitespaceRight(data []byte) []byte {
	var i int
	for i = len(data); i > 0; i-- {
		c := data[i-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' && c != '\f' {
			break
		}
	}
	return data[:i]
}

var hardLineBreakRe = regexp.MustCompile(`( {2,}|\\)\n *`)
var hardLineBreak = []byte("<br />\n")

func processHardLineBreaks(n *Node) *Node {
	if t, ok := n.Content().(*Text); ok {
		text := t.Content
		m := hardLineBreakRe.FindAllIndex(text, -1)
		if len(m) == 0 {
			return n
		}
		n = NewNode(nil)
		lineStart := 0
		for i := 0; i < len(m); i++ {
			n.AppendChild(NewNode(&Text{text[lineStart:m[i][0]]}))
			n.AppendChild(NewNode(&HardLineBreak{}))
			lineStart = m[i][1]
		}
		n.AppendChild(NewNode(&Text{text[lineStart:]}))
		return n
	} else {
		for child := n.FirstChild(); child != nil; child = child.Next() {
			processHardLineBreaks(child)
		}
		return n
	}
}

// collapseSpace collapses whitespace more or less the way a browser would:
// every run of space and newline characters is replaced by a single space.
// Other whitespace remains unaffected, but this is what the spec says for code
// spans.
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
