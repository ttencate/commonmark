package commonmark

type Inline interface {
}

type stringInline struct {
	content []byte
}
