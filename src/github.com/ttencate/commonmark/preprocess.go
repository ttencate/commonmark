package commonmark

import (
	"bufio"
	"bytes"
	"fmt"
)

// preprocess returns a new Document node with RawLine nodes as
// children, one for each input line.
func preprocess(data []byte) (*Node, error) {
	root := NewNode(&Document{})
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(scanLines)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)
		line = append(line, '\n')

		var indentation int
		for indentation := 0; indentation < len(line); indentation++ {
			if line[indentation] != ' ' {
				break
			}
		}

		root.AppendChild(NewNode(&RawLine{
			Content:           line,
			Indentation:       indentation,
			IsBlank:           line[indentation] == '\n',
			FirstNonSpaceChar: line[indentation],
		}))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return root, nil
}

// scanLines is a split function for bufio.Scanner that splits on CR, LF or
// CRLF pairs. We need this because bufio.ScanLines itself only does CR and
// CRLF.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' {
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[0:i], nil
			} else {
				return i + 1, data[0:i], nil
			}
		} else if data[i] == '\n' {
			return i + 1, data[0:i], nil
		}
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// tabsToSpaces returns a slice (possibly the same one) in which tabs have been
// replaced by up to 4 spaces, depending on the tab stop position.
//
// It does not modify the input slice; a copy is made if needed.
func tabsToSpaces(line []byte) []byte {
	const tabStop = 4

	var tabCount int
	for _, c := range line {
		if c == '\t' {
			tabCount++
		}
	}
	if tabCount == 0 {
		return line
	}

	output := make([]byte, 0, len(line)+3*tabCount)
	var runeCount int
	for _, c := range string(line) {
		if c == '\t' {
			numSpaces := tabStop - runeCount%tabStop
			spaces := bytes.Repeat([]byte{' '}, numSpaces)
			output = append(output, spaces...)
			runeCount += numSpaces
		} else {
			output = append(output, []byte(fmt.Sprintf("%c", c))...)
			runeCount++
		}
	}
	return output
}
