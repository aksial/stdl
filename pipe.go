package stdl

import (
	"io"
)

type pipe struct {
	io.ReadWriter
	r io.Reader
	w io.Writer
}

func Pipe() io.ReadWriter {
	p := new(pipe)
	p.r, p.w = io.Pipe()
	return p
}

func (p *pipe) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

func (p *pipe) Write(b []byte) (int, error) {
	return p.w.Write(b)
}
