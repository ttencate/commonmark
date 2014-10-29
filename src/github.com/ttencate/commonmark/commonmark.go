// Package commonmark provides functionality to convert CommonMark syntax to
// HTML.
package commonmark

import (
	"bytes"
	"unicode"
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
	doc, err := parse(data)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	blockToHTML(doc, &buffer)
	return buffer.Bytes(), nil
}

func parse(data []byte) (*document, error) {
	// See http://spec.commonmark.org/0.7/#appendix-a-a-parsing-strategy

	// Phase one: construct a tree of blocks, and store reference definitions.
	doc, err := parseBlocks(data)
	if err != nil {
		return nil, err
	}

	// Phase two: process inlines.
	processInlines(doc)

	return doc, nil
}

func parseBlocks(data []byte) (*document, error) {
	scanner := newScanner(data)
	doc := &document{}
	openBlocks := []Block{doc}
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')

		var openBlock Block
		for _, openBlock = range openBlocks {
			if _, ok := openBlock.(LeafBlock); ok {
				break
			}
		}

		leafBlock, ok := openBlock.(LeafBlock)
		if !ok {
			containerBlock := openBlock.(ContainerBlock)
			leafBlock = &paragraph{}
			containerBlock.AppendChild(leafBlock)
			openBlocks = append(openBlocks, leafBlock)
		}
		leafBlock.AppendLine(line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

func processInlines(b Block) {
	switch t := b.(type) {
	case *paragraph:
		// "Final spaces are stripped before inline parsing, so a paragraph that
		// ends with two or more spaces will not end with a hard line break."
		t.inlineContent = parseInlines(bytes.TrimRight(t.content, " "))
	}

	if container, ok := b.(ContainerBlock); ok {
		for _, child := range container.Children() {
			processInlines(child)
		}
	}
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
	for p.pos < len(p.data) {
		var inline Inline
		switch p.data[p.pos] {
		case '\n':
			hardBreak := false
			newlinePos := p.pos
			if p.pos >= 1 && p.data[p.pos-1] == '\\' {
				hardBreak = true
				p.pos--
			} else if p.pos >= 2 && p.data[p.pos-1] == ' ' && p.data[p.pos-2] == ' ' {
				hardBreak = true
				p.pos -= 2
			}

			// "Spaces at the end of the line [...] are removed."
			for p.pos > 0 && p.data[p.pos - 1] == ' ' {
				p.pos--
			}
			p.finalizeString()

			if hardBreak {
				inline = &hardLineBreak{}
			} else {
				inline = &softLineBreak{}
			}

			p.pos = newlinePos + 1
			// "Spaces at [...] the beginning of the next line are removed."
			for p.pos < len(p.data) && p.data[p.pos] == ' ' {
				p.pos++
			}
			p.resetString()
		case '`':
			// "A backtick string is a string of one or more backtick
			// characters (`) that is neither preceded nor followed by a
			// backtick."
			var numBackticks int
			for p.pos+numBackticks < len(p.data) && p.data[p.pos+numBackticks] == '`' {
				numBackticks++
			}
			closing := backtickStringIndex(p.data, p.pos+numBackticks, numBackticks)
			if closing == -1 {
				p.pos += numBackticks
				break
			}

			p.finalizeString()
			p.pos += numBackticks

			content := p.data[p.pos:closing]
			content = collapseSpace(bytes.TrimSpace(content))

			inline = &codeSpan{content}
			p.pos = closing + numBackticks
			p.resetString()
		default:
			p.pos++
		}

		if inline != nil {
			p.root.children = append(p.root.children, inline)
		}
	}
	p.finalizeString()
}

func backtickStringIndex(data []byte, start, length int) int {
	var count int
	for i := start; i < len(data); i++ {
		if data[i] == '`' {
			count++
			if count == length {
				if i+1 >= len(data) || data[i+1] != '`' {
					return i + 1 - count
				}
			}
		} else {
			count = 0
		}
	}
	return -1
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
