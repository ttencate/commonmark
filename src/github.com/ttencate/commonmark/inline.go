package commonmark

import (
	"bytes"
	"regexp"
	"strings"
)

// RawText is the NodeContent for text that still needs to go through inline
// parsing. Nodes of this type cannot have children.
type RawText struct {
	Content []byte
}

// Text is the NodeContent for text strings, sent to the output verbatim (but
// possibly with escaping). Nodes of this type cannot have children.
type Text struct {
	Content []byte
}

// LiteralText is the NodeContent for text that should not be touched during
// processing anymore.
type LiteralText struct {
	Content []byte
}

// BackslashEscape is the NodeContent for characters escaped with a backslash.
type BackslashEscape struct {
	Content []byte
}

// CodeSpan is the NodeContent for inline code spans.
type CodeSpan struct {
	Content []byte
}

// RawHTML is the NodeContent for raw HTML, sent to the output without any
// escaping.
type RawHTML struct {
	Content []byte
}

// Link is the NodeContent for links.
type Link struct {
	Href []byte
}

// Emphasis is the NodeContent for emphasis, typically rendered as italic text.
type Emphasis struct{}

// StrongEmphasis is the NodeContent for strong emphasis, typically rendered as
// bold text.
type StrongEmphasis struct{}

// HardLineBreak is the NodeContent for hard line breaks, rendered as <br />
// tags in HTML.
type HardLineBreak struct{}

// parseInlines processes inline elements on the given node. It is assumed to
// be a RawText node containing the given text.
func parseInlines(n *Node, text []byte) {
	// I can't find where the spec decrees this. But the reference
	// implementation does it this way:
	// https://github.com/jgm/CommonMark/blob/67619a5d5c71c44565a9a0413aaf78f9baece528/src/inlines.c#L183
	// Filed issue:
	// https://github.com/jgm/CommonMark/issues/176
	n.SetContent(&Text{trimWhitespaceRight(text)})

	applyRecursively(n, forTextNodes(processRawHTML))
	applyRecursively(n, forTextNodes(processCodeSpans))
	applyRecursively(n, forTextNodes(processAutolinks))
	applyRecursively(n, forTextNodes(processBackslashEscapes))
	applyRecursively(n, processEmphasis)
	applyRecursively(n, forTextNodes(processHardLineBreaks))
	applyRecursively(n, forTextNodes(processSoftLineBreaks))
}

var asciiPunct = []byte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~")

// isASCIIPunct mimics the behaviour of ispunct(3). It is neither a subset nor
// a superset of unicode.IsPunct; for instance, '`' is not considered
// punctuation by Unicode.
func isASCIIPunct(char byte) bool {
	// TODO speed this up using a []bool
	return bytes.IndexByte(asciiPunct, char) >= 0
}

// isASCIIAlNum mimics the behaviour of isalnum(3) for the C locale.
func isASCIIAlNum(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

// isASCIISpace mimics the behaviour of isspace(3) for the C locale.
func isASCIISpace(char byte) bool {
	return char == ' ' || char == '\n' || char == '\r' || char == '\t' || char == '\f'
}

// trimWhitespaceRight returns a subslice where all trailing whitespace (ASCII
// only) is removed.
func trimWhitespaceRight(data []byte) []byte {
	var i int
	for i = len(data); i > 0; i-- {
		c := data[i-1]
		if !isASCIISpace(c) {
			break
		}
	}
	return data[:i]
}

// applyRecursively applies the given function to each node in the tree in
// turn in a depth-first way, children first, then their parent.
//
// The function may modify the given node and its children, but must leave this
// node in place and may not modify the tree above or around it. Note that any
// new children will not be visited.
func applyRecursively(n *Node, f func(*Node)) {
	// TODO add maximum recursion depth to prevent stack overflow (possible
	// denial-of-service attack vector)
	for child := n.FirstChild(); child != nil; child = child.Next() {
		applyRecursively(child, f)
	}
	f(n)
}

// forTextNodes returns a function that can be given to applyRecursively. The
// returned function applies f to Text nodes, passing in the text content.
func forTextNodes(f func(*Node, []byte)) func(*Node) {
	return func(n *Node) {
		if t, ok := n.Content().(*Text); ok {
			f(n, t.Content)
		}
	}
}

func processBackslashEscapes(n *Node, text []byte) {
	doneAnySplits := false
	for i := 0; i < len(text)-1; {
		if text[i] == '\\' && isASCIIPunct(text[i+1]) {
			if !doneAnySplits {
				n.AppendChild(NewNode(&Text{text}))
				n.SetContent(nil)
				n = n.FirstChild()
				doneAnySplits = true
			}
			n = splitTextNode(n, i, i+2)
			n.InsertBefore(NewNode(&BackslashEscape{text[i+1 : i+2]}))
			text = n.Content().(*Text).Content
			i = 0
		} else {
			i++
		}
	}
}

func processCodeSpans(n *Node, text []byte) {
	// "A backtick string is a string of one or more backtick characters (`)
	// that is neither preceded nor followed by a backtick. A code span begins
	// with a backtick string and ends with a backtick string of equal length.
	// The contents of the code span are the characters between the two
	// backtick strings, with leading and trailing spaces and newlines removed,
	// and consecutive spaces and newlines collapsed to single spaces.

	textNode := NewNode(&Text{text})
	n.AppendChild(textNode)
	n.SetContent(nil)

	// Store previously seen, non-matched backtick strings by length.
	// TODO This is buggy for overlapping spans, use a stack. Consider this case:
	// ``foo`bar``biz`
	// The second `` will match the first, and take the single ` up into the
	// code span, but the second ` will still try to match it.
	for textNode != nil {
		var backtickStringsByLength = make(map[int]int)
		for i := 0; i < len(text); i++ {
			if text[i] != '`' {
				continue
			}
			if i > 0 && text[i-1] == '\\' {
				continue
			}
			// Determine length of backtick string.
			numBackticks := 0
			for ; i < len(text) && text[i] == '`'; i++ {
				numBackticks++
			}
			// At end of backtick string. See if we have seen a matching one.
			if opener, found := backtickStringsByLength[numBackticks]; found {
				// Previous string of matching length exists. Extract code
				// span.
				code := text[opener+numBackticks : i-numBackticks]
				code = bytes.Trim(code, " \n")
				code = collapseSpace(code)
				textNode = splitTextNode(textNode, opener, i)
				textNode.InsertBefore(NewNode(&CodeSpan{code}))
				// Code spans are never nested, so any subsequent string can
				// never match any seen before.
				// The spec is ambiguous about what happens in this case, e.g.:
				// `` `foo` `` can become either
				// <code>`foo`</code>
				// or
				// `` <code>foo</code> ``
				// TODO file a spec bug about this when I can get online again
				break
			} else {
				// No previous string of this length encountered. Store as
				// potential opener.
				backtickStringsByLength[numBackticks] = i - numBackticks
			}
		}
		// At the end of the last text node: we are done.
		break
	}
}

// splitTextNode splits off another node on the right of a Text node, possibly
// omitting part of the intermediate text. It returns the newly created node.
func splitTextNode(n *Node, endOfFirst, startOfSecond int) *Node {
	text := n.Content().(*Text).Content
	n.SetContent(&Text{text[:endOfFirst]})
	right := NewNode(&Text{text[startOfSecond:]})
	n.InsertAfter(right)
	return right
}

func processEmphasis(n *Node) {
	// TODO handle when the root node is text. Does not happen thanks to code
	// span parsing, but should not rely on it.
	type stackEntry struct {
		delimChar   byte
		delimsStart int
		numDelims   int
		textNode    *Node
	}
	// TODO get _foo *bar_ biz* added to the spec (symmetrical case is there)
	var stack []stackEntry
	for child := n.FirstChild(); child != nil; child = child.Next() {
		var text []byte
		if textContent, ok := child.Content().(*Text); ok {
			text = textContent.Content
		} else {
			// We have already been recursively applied to this child. Nothing
			// to do.
			continue
		}
		for i := 0; i < len(text); {
			c := text[i]
			if c == '*' || c == '_' {
				// Find the end of this delimiter string.
				numDelims := 0
				delimsStart := i
				for ; i < len(text) && text[i] == c; i++ {
					numDelims++
				}
				// Ignore any string of more than 3 delimiters entirely.
				if numDelims > 3 {
					continue
				}
				// See if this delimiter string can close emphasis.
				if (delimsStart == 0 || !isASCIISpace(text[delimsStart-1])) &&
					(c != '_' || i == len(text) || !isASCIIAlNum(text[i])) {
					// Walk the stack until we find a suitable opening delimiter.
					var j int
					var entry stackEntry
					for j = len(stack) - 1; j >= 0; j-- {
						entry = stack[j]
						if entry.delimChar == c {
							break
						}
					}
					if j >= 0 {
						// Consume as many delimiter characters as we can:
						// either 1 or 2.
						consumeDelims := 1
						if numDelims == 3 && entry.numDelims == 3 {
							// "An interpretation <strong><em>...</em></strong>
							// is always preferred to
							// <em><strong>..</strong></em>."
							consumeDelims = 1
						} else if numDelims >= 2 && entry.numDelims >= 2 {
							// "An interpretation <strong>...</strong> is
							// always preferred to <em><em>...</em></em>."
							consumeDelims = 2
						}
						// Split the text nodes containing the delimiters,
						// omitting the innermost used delimiter characters
						// themselves. Split the closer node first in case it
						// is the same as the opener node.
						nodeAfterCloser := splitTextNode(child, delimsStart, delimsStart+consumeDelims)
						splitTextNode(entry.textNode, entry.delimsStart+entry.numDelims-consumeDelims, entry.delimsStart+entry.numDelims)
						// After the opener node, insert a container for the
						// emphasis span, and shove all subsequent siblings
						// underneath it.
						var emphasisNode *Node
						if consumeDelims == 1 {
							emphasisNode = NewNode(&Emphasis{})
						} else {
							emphasisNode = NewNode(&StrongEmphasis{})
						}
						entry.textNode.InsertAfter(emphasisNode)
						for {
							sibling := emphasisNode.Next()
							if sibling == nodeAfterCloser {
								break
							}
							sibling.Remove()
							emphasisNode.AppendChild(sibling)
						}
						// Ensure we continue in the right place: with the next
						// sibling of the newly created emphasis node.
						child = emphasisNode
						// Trim the stack.
						stack = stack[:j]
						// If not all opener delimiters were consumed, push the
						// entry back onto the stack. Its node pointer is still
						// valid.
						entry.numDelims -= consumeDelims
						if entry.numDelims > 0 {
							stack = append(stack, entry)
						}
						// And restart the inner loop so it picks up the right
						// indices for the subsequent node.
						break
					}
				}
				// See if this delimiter string can open emphasis.
				if (i == len(text) || !isASCIISpace(text[i])) &&
					(c != '_' || delimsStart == 0 || !isASCIIAlNum(text[delimsStart-1])) {
					// Push it onto the stack.
					stack = append(stack, stackEntry{c, delimsStart, numDelims, child})
				}
			} else {
				i++
			}
		}
	}
}

// "A tag name consists of an ASCII letter followed by zero or more ASCII
// letters or digits."
var tagName = `[a-zA-Z][a-zA-Z0-9]*`

// "An attribute value specification consists of optional whitespace, a = character, optional whitespace, and an attribute value.
// "An attribute value consists of an unquoted attribute value, a single-quoted
// attribute value, or a double-quoted attribute value.
// "An unquoted attribute value is a nonempty string of characters not
// including spaces, ", ', =, <, >, or `.
// "A single-quoted attribute value consists of ', zero or more characters not
// including ', and a final '.
// "A double-quoted attribute value consists of ", zero or more characters not
// including ", and a final "."
var attributeValueSpecification = `\s*=\s*([^ "'=<>` + "`" + `]+|'[^']*'|"[^"]*"`

// "An attribute consists of whitespace, an attribute name, and an optional
// attribute value specification."
var attribute = `\s+[a-zA-Z_:][a-zA-Z0-9_.:-]*(` + attributeValueSpecification + `))?`

// "An HTML tag consists of an open tag, a closing tag, an HTML comment, a
// processing instruction, an element type declaration, or a CDATA section."
var rawHTMLRe = regexp.MustCompile("(?s:" + strings.Join([]string{
	// "An open tag consists of a < character, a tag name, zero or more
	// attributes, optional whitespace, an optional / character, and a >
	// character."
	`<` + tagName + `(` + attribute + `)*\s*/?>`,
	// "A closing tag consists of the string </, a tag name, optional
	// whitespace, and the character >."
	`</` + tagName + `\s*>`,
	// "An HTML comment consists of the string <!--, a string of characters not
	// including the string --, and the string -->."
	`<!--(-?[^-])*-?-->`,
	// "A processing instruction consists of the string <?, a string of
	// characters not including the string ?>, and the string ?>."
	`<\?.*?\?>`,
	// "A declaration consists of the string <!, a name consisting of one or
	// more uppercase ASCII letters, whitespace, a string of characters not
	// including the character >, and the character >.
	`<![A-Z]+\s+.+?>`,
	// "A CDATA section consists of the string <![CDATA[, a string of
	// characters not including the string ]]>, and the string ]]>."
	`<!\[CDATA\[.*?\]\]>`}, "|") + ")")

func processRawHTML(n *Node, text []byte) {
	m := rawHTMLRe.FindAllIndex(text, -1)
	if len(m) == 0 {
		return
	}
	n.SetContent(nil)
	textStart := 0
	for i := 0; i < len(m); i++ {
		if m[i][0] > 0 && text[m[i][0]-1] == '\\' {
			continue
		}
		n.AppendChild(NewNode(&Text{text[textStart:m[i][0]]}))
		n.AppendChild(NewNode(&RawHTML{text[m[i][0]:m[i][1]]}))
		textStart = m[i][1]
	}
	n.AppendChild(NewNode(&Text{text[textStart:]}))
}

// "The following schemes are recognized (case-insensitive): ..."
var scheme = `(?i:coap|doi|javascript|aaa|aaas|about|acap|cap|cid|crid|data|dav|dict|dns|file|ftp|geo|go|gopher|h323|http|https|iax|icap|im|imap|info|ipp|iris|iris\.beep|iris\.xpc|iris\.xpcs|iris\.lwz|ldap|mailto|mid|msrp|msrps|mtqp|mupdate|news|nfs|ni|nih|nntp|opaquelocktoken|pop|pres|rtsp|service|session|shttp|sieve|sip|sips|sms|snmp|soap\.beep|soap\.beeps|tag|tel|telnet|tftp|thismessage|tn3270|tip|tv|urn|vemmi|ws|wss|xcon|xcon-userid|xmlrpc\.beep|xmlrpc\.beeps|xmpp|z39\.50r|z39\.50s|adiumxtra|afp|afs|aim|apt|attachment|aw|beshare|bitcoin|bolo|callto|chrome|chrome-extension|com-eventbrite-attendee|content|cvs|dlna-playsingle|dlna-playcontainer|dtn|dvb|ed2k|facetime|feed|finger|fish|gg|git|gizmoproject|gtalk|hcp|icon|ipn|irc|irc6|ircs|itms|jar|jms|keyparc|lastfm|ldaps|magnet|maps|market|message|mms|ms-help|msnim|mumble|mvn|notes|oid|palm|paparazzi|platform|proxy|psyc|query|res|resource|rmi|rsync|rtmp|secondlife|sftp|sgn|skype|smb|soldat|spotify|ssh|steam|svn|teamspeak|things|udp|unreal|ut2004|ventrilo|view-source|webcal|wtai|wyciwyg|xfire|xri|ymsgr)`

// "An absolute URI, for these purposes, consists of a scheme followed by a
// colon (:) followed by zero or more characters other than ASCII whitespace
// and control characters, <, and >."
var absoluteURI = scheme + ":[^[:space:][:cntrl:]<>]*"

// "An email address, for these purposes, is anything that matches the non-normative regex from the HTML5 spec:"
var emailAddress = "[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*"

// "A URI autolink consists of <, followed by an absolute URI not containing <,
// followed by >."
// "An email autolink consists of <, followed by an email address, followed by
// >. The link’s label is the email address, and the URL is mailto: followed by
// the email address."
var autolinkRe = regexp.MustCompile(`<(?:(` + absoluteURI + `)|(` + emailAddress + `))>`)

var mailto = []byte("mailto:")

func processAutolinks(n *Node, text []byte) {
	m := autolinkRe.FindAllSubmatchIndex(text, -1)
	if len(m) == 0 {
		return
	}
	n.SetContent(nil)
	textStart := 0
	for i := 0; i < len(m); i++ {
		// TODO there is no test in the spec for escaped \<autolink>; file bug
		if m[i][0] > 0 && text[m[i][0]-1] == '\\' {
			continue
		}
		n.AppendChild(NewNode(&Text{text[textStart:m[i][0]]}))
		var uri []byte
		var content []byte
		if m[i][2] >= 0 {
			uri = text[m[i][2]:m[i][3]]
			content = text[m[i][2]:m[i][3]]
		} else {
			uri = append(mailto, text[m[i][4]:m[i][5]]...)
			content = text[m[i][4]:m[i][5]]
		}
		linkNode := NewNode(&Link{uri})
		linkNode.AppendChild(NewNode(&LiteralText{content}))
		n.AppendChild(linkNode)
		textStart = m[i][1]
	}
	n.AppendChild(NewNode(&Text{text[textStart:]}))
}

var hardLineBreakRe = regexp.MustCompile(`( {2,}|\\)\n *`)

func processHardLineBreaks(n *Node, text []byte) {
	m := hardLineBreakRe.FindAllIndex(text, -1)
	if len(m) == 0 {
		return
	}
	n.SetContent(nil)
	lineStart := 0
	for i := 0; i < len(m); i++ {
		n.AppendChild(NewNode(&Text{text[lineStart:m[i][0]]}))
		n.AppendChild(NewNode(&HardLineBreak{}))
		lineStart = m[i][1]
	}
	n.AppendChild(NewNode(&Text{text[lineStart:]}))
}

var softLineBreakRe = regexp.MustCompile(` *\n *`)
var softLineBreak = []byte("\n")

func processSoftLineBreaks(n *Node, text []byte) {
	text = softLineBreakRe.ReplaceAll(text, softLineBreak)
	n.SetContent(&Text{text})
}

// collapseSpace collapses whitespace more or less the way a browser would:
// every run of space and newline characters is replaced by a single space.
// Other whitespace remains unaffected, but this is what the spec says for code
// spans.
// TODO if nothing was collapsed, don't allocate a new slice
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
