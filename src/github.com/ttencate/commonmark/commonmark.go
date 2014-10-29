// Package commonmark provides functionality to convert CommonMark syntax to
// HTML.
package commonmark

// ToHTML converts a string formatted in CommonMark into the corresponding HTML
// string.
//
// Note that it might contain unsafe tags (e.g. <script>); if you are accepting
// untrusted user input, you must run the output through a sanitizer before
// sending it to a browser.
func ToHTML(markdown string) string {
  return markdown
}
