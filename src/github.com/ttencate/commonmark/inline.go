package commonmark

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
