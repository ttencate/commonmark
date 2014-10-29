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
		out.Write(t.content)
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
		out.Write(t.content)
		io.WriteString(out, "</code>")
	default:
		log.Panicf("no HTML converter registered for Inline type %T", i)
	}
}
