package commonmark

import (
	"bytes"
	"fmt"
	"strconv"
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
	for p.pos < len(p.data) {
		var inline Inline
		switch p.data[p.pos] {
		case '\n':
			hardBreak := false
			newlinePos := p.pos
			// "A line break (not in a code span or HTML tag) that is preceded
			// by two or more spaces is parsed as a hard line break."
			if p.pos >= 2 && p.data[p.pos-1] == ' ' && p.data[p.pos-2] == ' ' {
				hardBreak = true
				p.pos -= 2
			}
			// "For a more visible alternative, a backslash before the newline
			// may be used instead of two spaces."
			if p.pos >= 1 && p.data[p.pos-1] == '\\' {
				hardBreak = true
				p.pos--
			}

			// "Spaces at the end of the line [...] are removed."
			for p.pos > 0 && p.data[p.pos-1] == ' ' {
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
		case '\\':
			// "Backslashes before other characters are treated as literal backslashes."
			if p.pos+1 >= len(p.data) || !isASCIIPunct(p.data[p.pos+1]) {
				p.pos++
				break
			}

			// "Any ASCII punctuation character may be backslash-escaped."
			p.finalizeString()
			p.pos++
			inline = &stringInline{p.data[p.pos : p.pos+1]}
			p.pos++
			p.resetString()
		case '&':
			// "[A]ll valid HTML entities in any context are recognized as such
			// and converted into unicode characters before they are stored in
			// the AST."
			semicolon := bytes.IndexByte(p.data[p.pos+1:], ';')
			// "Although HTML5 does accept some entities without a trailing
			// semicolon (such as &copy), these are not recognized as entities
			// here, because it makes the grammar too ambiguous."
			if semicolon < 0 {
				p.pos++
				break
			}
			semicolon += p.pos + 1
			entity := string(p.data[p.pos+1 : semicolon])
			var codepoints string

			if len(entity) > 0 {
				if entity[0] == '#' {
					if len(entity) > 1 {
						if entity[1] == 'x' || entity[1] == 'X' {
							// "Hexadecimal entities consist of &# + either X or x + a
							// string of 1-8 hexadecimal digits + ;."
							if codepoint, err := strconv.ParseUint(entity[2:], 16, 32); err == nil {
								codepoints = fmt.Sprintf("%c", codepoint)
							}
						} else {
							// "Decimal entities consist of &# + a string of 1–8 arabic
							// digits + ;. Again, these entities need to be recognised and
							// tranformed into their corresponding UTF8 codepoints. Invalid
							// Unicode codepoints will be written as the “unknown
							// codepoint” character (0xFFFD)."
							if codepoint, err := strconv.ParseUint(entity[1:], 10, 32); err == nil {
								codepoints = fmt.Sprintf("%c", codepoint)
							}
						}
					}
				} else {
					// "Named entities consist of & + any of the valid HTML5 entity names + ;."
					codepoints = htmlEntities[entity]
				}
			}

			if len(codepoints) == 0 {
				p.pos++
				break
			}

			p.finalizeString()
			inline = &stringInline{[]byte(codepoints)}
			p.pos = semicolon + 1
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

var asciiPunct = []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

func isASCIIPunct(char byte) bool {
	return bytes.IndexByte(asciiPunct, char) >= 0
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
