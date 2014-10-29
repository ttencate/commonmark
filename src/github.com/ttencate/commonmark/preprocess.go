package commonmark

import (
	"bufio"
	"bytes"
)

// newScanner returns a new bufio.Scanner suitable for reading lines.
func newScanner(data []byte) *bufio.Scanner {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(scanLines)
	return scanner
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
	for remaining := line; len(remaining) > 0; {
		i := bytes.IndexByte(remaining, '\n')
		if i == -1 {
			break
		}
		remaining = remaining[i+1:]
		tabCount++
	}
	if tabCount == 0 {
		return line
	}

	output := make([]byte, 0, len(line)+4*tabCount)
	for i := 0; i < len(line); i++ {
		if line[i] == '\t' {

			spaces := bytes.Repeat([]byte{' '}, tabStop-i%tabStop)
			output = append(output, spaces...)
		} else {
			output = append(output, line[i])
		}
	}
	return output
}
