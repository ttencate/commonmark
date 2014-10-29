// Package commonmark provides functionality to convert CommonMark syntax to
// HTML.
package commonmark

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
	scanner := newScanner(data)
	var html []byte
	for scanner.Scan() {
		line := scanner.Bytes()
		line = tabsToSpaces(line)

		html = append(html, line...)
		html = append(html, '\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return html, nil
}
