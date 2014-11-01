package commonmark

import (
	"bytes"
	"unicode"
)

type Inline interface {
}

type stringInline struct {
	content []byte
}

type softLineBreak struct{}

type hardLineBreak struct{}

type codeSpan struct {
	content []byte
}

type multipleInline struct {
	children []Inline
}

type inlineParser struct {
	data        []byte
	pos         int
	stringStart int

	root *multipleInline
}

func parseInlines(data []byte) Inline {
	// I can't find where the spec decrees this. But the reference
	// implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	data = bytes.TrimRightFunc(data, unicode.IsSpace)

	parser := inlineParser{
		data: data,
		root: &multipleInline{},
	}
	parser.parse()
	return parser.root
}

func (p *inlineParser) parse() {
}

var asciiPunct = []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

func isASCIIPunct(char byte) bool {
	return bytes.IndexByte(asciiPunct, char) >= 0
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

func (p *inlineParser) resetString() {
	p.stringStart = p.pos
}

func (p *inlineParser) finalizeString() {
	if p.stringStart >= p.pos {
		return
	}
	str := p.data[p.stringStart:p.pos]
	p.root.children = append(p.root.children, &stringInline{str})
}
