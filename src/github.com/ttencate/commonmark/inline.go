package commonmark

import (
	"bytes"
	"regexp"
	"strings"
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

// RawHTML is the NodeContent for raw HTML, sent to the output without any
// escaping.
type RawHTML struct {
	Content []byte
}

// HardLineBreak is the NodeContent for hard line breaks, rendered as <br />
// tags in HTML.
type HardLineBreak struct{}

// parseInlines processes inline elements on the given node. It is assumed to
// be a RawText node containing the given text.
func parseInlines(n *Node, text []byte) {
	// I can't find where the spec decrees this. But the reference
	// implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	// Filed issue:
	// https://github.com/jgm/CommonMark/issues/176
	n.SetContent(&Text{trimWhitespaceRight(text)})

	applyRecursively(n, toText(processRawHTML))
	applyRecursively(n, toText(processHardLineBreaks))
	applyRecursively(n, toText(processSoftLineBreaks))
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

// applyRecursively applies the given function to each node in the tree in
// turn. Parents first, then their children in order. If the function returns
// false for any node, recursion does not descend into that node's children.
//
// The function may modify the given node and its children, but must leave this
// node in place and may not modify the tree above or around it.
func applyRecursively(n *Node, f func(*Node) bool) {
	if f(n) {
		for child := n.FirstChild(); child != nil; child = child.Next() {
			applyRecursively(child, f)
		}
	}
}

// toText returns a function that can be given to applyRecursively. The
// returned function applies f to Text nodes, passing in the text content, and
// returning false for text nodes only.
func toText(f func(*Node, []byte)) func(*Node) bool {
	return func(n *Node) bool {
		if t, ok := n.Content().(*Text); ok {
			f(n, t.Content)
			return false
		}
		return true
	}
}

// "A tag name consists of an ASCII letter followed by zero or more ASCII
// letters or digits."
var tagName = `[a-zA-Z][a-zA-Z0-9]*`

// "An attribute value specification consists of optional whitespace, a = character, optional whitespace, and an attribute value.
// "An attribute value consists of an unquoted attribute value, a single-quoted
// attribute value, or a double-quoted attribute value.
// "An unquoted attribute value is a nonempty string of characters not
// including spaces, ", ', =, <, >, or `.
// "A single-quoted attribute value consists of ', zero or more characters not
// including ', and a final '.
// "A double-quoted attribute value consists of ", zero or more characters not
// including ", and a final "."
var attributeValueSpecification = `\s*=\s*([^ "'=<>` + "`" + `]+|'[^']*'|"[^"]*"`
var attribute = `\s+[a-zA-Z_:][a-zA-Z0-9_.:-]*(` + attributeValueSpecification + `))?`

// "An HTML tag consists of an open tag, a closing tag, an HTML comment, a
// processing instruction, an element type declaration, or a CDATA section."
var rawHTMLRe = regexp.MustCompile("(?s:" + strings.Join([]string{
	// "An open tag consists of a < character, a tag name, zero or more
	// attributes, optional whitespace, an optional / character, and a >
	// character."
	`<` + tagName + `(` + attribute + `)*\s*/?>`,
	// "A closing tag consists of the string </, a tag name, optional
	// whitespace, and the character >."
	`</` + tagName + `\s*>`,
	// "An HTML comment consists of the string <!--, a string of characters not
	// including the string --, and the string -->."
	`<!--(-?[^-])*-?-->`,
	// "A processing instruction consists of the string <?, a string of
	// characters not including the string ?>, and the string ?>."
	`<\?.*?\?>`,
	// "A declaration consists of the string <!, a name consisting of one or
	// more uppercase ASCII letters, whitespace, a string of characters not
	// including the character >, and the character >.
	`<![A-Z]+\s+.+?>`,
	// "A CDATA section consists of the string <![CDATA[, a string of
	// characters not including the string ]]>, and the string ]]>."
	`<!\[CDATA\[.*?\]\]>`}, "|") + ")")

func processRawHTML(n *Node, text []byte) {
	m := rawHTMLRe.FindAllIndex(text, -1)
	if len(m) == 0 {
		return
	}
	n.SetContent(nil)
	textStart := 0
	for i := 0; i < len(m); i++ {
		n.AppendChild(NewNode(&Text{text[textStart:m[i][0]]}))
		n.AppendChild(NewNode(&RawHTML{text[m[i][0]:m[i][1]]}))
		textStart = m[i][1]
	}
	n.AppendChild(NewNode(&Text{text[textStart:]}))
}

var hardLineBreakRe = regexp.MustCompile(`( {2,}|\\)\n *`)

func processHardLineBreaks(n *Node, text []byte) {
	m := hardLineBreakRe.FindAllIndex(text, -1)
	if len(m) == 0 {
		return
	}
	n.SetContent(nil)
	lineStart := 0
	for i := 0; i < len(m); i++ {
		n.AppendChild(NewNode(&Text{text[lineStart:m[i][0]]}))
		n.AppendChild(NewNode(&HardLineBreak{}))
		lineStart = m[i][1]
	}
	n.AppendChild(NewNode(&Text{text[lineStart:]}))
}

var softLineBreakRe = regexp.MustCompile(` *\n *`)
var softLineBreak = []byte("\n")

func processSoftLineBreaks(n *Node, text []byte) {
	text = softLineBreakRe.ReplaceAll(text, softLineBreak)
	n.SetContent(&Text{text})
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
