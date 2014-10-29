package commonmark

import (
	"io"
	"log"
)

func blockToHTML(b Block, out io.Writer) {
	// Why not simply a method on Block? Extensibility: we want to support
	// other (pluggable) output types than HTML, and also custom Block types.
	switch t := b.(type) {
	case *document:
		for _, child := range t.Children() {
			blockToHTML(child, out)
		}
	case *paragraph:
		io.WriteString(out, "<p>")
		inlineToHTML(t.inlineContent, out)
		io.WriteString(out, "</p>\n")
	default:
		log.Panicf("no HTML converter registered for Block type %T", b)
	}
}

func inlineToHTML(i Inline, out io.Writer) {
	switch t := i.(type) {
	case *stringInline:
		writeEscaped(t.content, out)
	case *multipleInline:
		for _, child := range t.children {
			inlineToHTML(child, out)
		}
	case *softLineBreak:
		io.WriteString(out, "\n")
	case *hardLineBreak:
		io.WriteString(out, "<br />\n")
	case *codeSpan:
		io.WriteString(out, "<code>")
		writeEscaped(t.content, out)
		io.WriteString(out, "</code>")
	default:
		log.Panicf("no HTML converter registered for Inline type %T", i)
	}
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
