package commonmark

import (
	"io"
	"log"
)

func toHTML(b Block, out io.Writer) {
	// Why not simply a method on Block? Extensibility: we want to support
	// other (pluggable) output types than HTML, and also custom Block types.
	switch t := b.(type) {
	case *document:
		for _, child := range t.Children() {
			toHTML(child, out)
		}
	case *paragraph:
		io.WriteString(out, "<p>")
		out.Write(t.content)
		io.WriteString(out, "</p>\n")
	default:
		log.Panicf("no HTML converter registered for Block type %T", b)
	}
}
