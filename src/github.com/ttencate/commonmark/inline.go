package commonmark

type Inline interface {
}

type stringInline struct {
	content []byte
}

type softLineBreak struct{}

type multipleInline struct {
	children []Inline
}
