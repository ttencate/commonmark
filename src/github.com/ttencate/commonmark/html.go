package commonmark

import (
	"io"
)

// ToHTML converts the given parse tree or fragment to HTML and writes it to
// the given output stream.
func ToHTML(n *Node, out io.Writer) {
	content := n.Content()
	startHTML(content)
	contentHTML(content)
	for child := n.FirstChild(); child != nil; child = child.Next() {
		ToHTML(child, out)
	}
	endHTML(content)
}

func startHTML(c NodeContent) {
}

func endHTML(c NodeContent) {
}

func contentHTML(c NodeContent) {
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
