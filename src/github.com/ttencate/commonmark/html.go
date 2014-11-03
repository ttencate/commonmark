package commonmark

import (
	"io"
)

// ToHTML converts the given parse tree or fragment to HTML and writes it to
// the given output stream.
func ToHTML(n *Node, out io.Writer) {
	content := n.Content()
	startHTML(content, out)
	wroteContent := contentHTML(content, out)
	if wroteContent && n.FirstChild() != nil {
		panic("node that wrote content cannot have children")
	}
	for child := n.FirstChild(); child != nil; child = child.Next() {
		ToHTML(child, out)
	}
	endHTML(content, out)
}

func startHTML(c NodeContent, out io.Writer) {
	switch c.(type) {
	case *CodeSpan:
		io.WriteString(out, "<code>")
	case *Emphasis:
		io.WriteString(out, "<em>")
	case *StrongEmphasis:
		io.WriteString(out, "<strong>")
	case *Paragraph:
		io.WriteString(out, "<p>")
	}
}

func endHTML(c NodeContent, out io.Writer) {
	switch c.(type) {
	case *CodeSpan:
		io.WriteString(out, "</code>")
	case *Emphasis:
		io.WriteString(out, "</em>")
	case *StrongEmphasis:
		io.WriteString(out, "</strong>")
	case *Paragraph:
		io.WriteString(out, "</p>\n")
	}
}

func contentHTML(c NodeContent, out io.Writer) bool {
	switch t := c.(type) {
	case *Text:
		writeEscaped(t.Content, out)
	case *CodeSpan:
		writeEscaped(t.Content, out)
	case *RawHTML:
		out.Write(t.Content)
	case *HardLineBreak:
		io.WriteString(out, "<br />\n")
	case *RawText:
		panic("raw text found in final parse tree")
	case *RawLine:
		panic("raw lines found in final parse tree")
	default:
		return false
	}
	return true
}

var escapeMap = map[byte]string{
	'"': "&quot;",
	'&': "&amp;",
	'<': "&lt;",
	'>': "&gt;",
}

func writeEscaped(data []byte, out io.Writer) {
	// "Conforming implementations that target HTML don’t need to generate
	// entities for all the valid named entities that exist, with the exception
	// of " (&quot;), & (&amp;), < (&lt;) and > (&gt;), which always need to be
	// written as entities for security reasons."
	var start int
	for i, c := range data {
		if escaped, ok := escapeMap[c]; ok {
			out.Write(data[start:i])
			io.WriteString(out, escaped)
			start = i + 1
		}
	}
	out.Write(data[start:])
}
