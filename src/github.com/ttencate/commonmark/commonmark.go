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

func parseInlines(data []byte) Inline {
	// I can't find where the spec decrees this. But the reference implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	data = bytes.TrimRightFunc(data, unicode.IsSpace)

	root := &multipleInline{}
	for len(data) > 0 {
		var inline Inline
		inline, data = parseInline(data)
		root.children = append(root.children, inline)
	}
	return root
}

func parseInline(data []byte) (Inline, []byte) {
	switch data[0] {
	case '\n':
		pos := 1
		// "Spaces at [...] the beginning of the next line are removed."
		for pos < len(data) && data[pos] == ' ' {
			pos++
		}
		return &softLineBreak{}, data[pos:]
	default:
		nextSpecialChar := bytes.IndexAny(data, "\n")
		if nextSpecialChar < 0 {
			nextSpecialChar = len(data)
		}
		// "Spaces at the end of the line [...] are removed."
		str := bytes.TrimRight(data[:nextSpecialChar], " ")
		return &stringInline{str}, data[nextSpecialChar:]
	}
}
