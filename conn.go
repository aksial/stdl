package stdl

import (
	"context"
	"encoding/hex"
	"io"
	"log"
	"net"
)

type input struct {
	io.ReadWriteCloser
	r io.Reader
	w io.WriteCloser

	c *conn
}

func newInput(c *conn, r io.Reader) *input {
	p := new(input)
	p.r, p.w = io.Pipe()
	p.c = c
	if r != nil {
		go func() {
			for {
				if _, err := io.Copy(p.w, r); err != nil {
					p.c.errorLogger.Print(err)
					return
				}
			}
		}()
	}
	return p
}

func (p *input) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if err != nil {
		p.c.errorLogger.Print(err)
	}
	if n > 0 {
		p.c.eventLogger.Printf("read %db:\n%s", n, hex.Dump(b[:n]))
	}
	return n, err
}

func (p *input) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if err != nil {
		p.c.errorLogger.Print(err)
	}
	if n > 0 {
		p.c.eventLogger.Printf("receive %db:\n%s", n, hex.Dump(b[:n]))
	}
	return n, err
}

func (p *input) Close() error {
	return nil
}

type conn struct {
	net.Conn
	ctx context.Context
	in  io.ReadWriter
	out io.Writer

	eventLogger *log.Logger
	errorLogger *log.Logger
}

func newConn(ctx context.Context, p io.ReadWriter) (c *conn, err error) {
	c = new(conn)
	c.ctx = ctx
	c.in = p //newInput(c, p)
	c.out = p
	if err != nil {
		return
	}

	c.eventLogger = eventLogger
	c.errorLogger = eventLogger

	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	ch := make(chan error)
	go func() {
		n, err = c.in.Read(b)
		ch <- err
	}()
	select {
	case <-c.ctx.Done():
		err = ErrContextCanceled
		return
	case err = <-ch:
	}
	return
}

func (c *conn) Write(b []byte) (t int, err error) {
	ch := make(chan error)

	go func() {
		for t < len(b) {
			n, err := c.out.Write(b[t:])
			c.eventLogger.Printf("wrote %db:\n%s", n, hex.Dump(b[t:t+n]))
			t += n
			if err != nil {
				ch <- err
				break
			}
		}
		ch <- nil
	}()

	select {
	case <-c.ctx.Done():
		if ctxCause := context.Cause(c.ctx); ctxCause != nil {
			err = ctxCause
			return
		}
		err = ErrContextCanceled
		return
	case err = <-ch:
	}
	return t, err
}

func (c *conn) Close() error {
	//defer c.in.Close()
	dc, ok := c.ctx.Value("disconnect").(func(context.Context))
	if ok {
		go dc(c.ctx)
	}
	return nil
}

func (c *conn) setEventLogger(l *log.Logger) {
	c.eventLogger = l
}

func (c *conn) setErrorLogger(l *log.Logger) {
	c.errorLogger = l
}

func (c *conn) RemoteAddr() net.Addr {
	return c
}

func (c *conn) LocalAddr() net.Addr {
	return c
}

func (c *conn) Network() string {
	return "io"
}

func (c *conn) String() string {
	return "io"
}
