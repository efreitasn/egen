package egen

type latexImageGenerator interface {
	SetDirPath(string) error
	SVGBlock([]byte) ([]byte, error)
	SVGInline([]byte) ([]byte, error)
}
