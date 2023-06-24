package stdl

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
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
		p.c.eventLogger.Printf("%s reads %db:\n%s", p.c.id(), n, hex.Dump(b[:n]))
	}
	return n, err
}

func (p *input) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if err != nil {
		p.c.errorLogger.Print(err)
	}
	if n > 0 {
		p.c.eventLogger.Printf("%s receives %db:\n%s", p.c.id(), n, hex.Dump(b[:n]))
	}
	return n, err
}

func (p *input) Close() error {
	return nil
}

type conn struct {
	net.Conn
	ctx context.Context
	in  io.ReadWriteCloser
	out io.Writer
	idb []byte

	eventLogger *log.Logger
	errorLogger *log.Logger
}

func newConn(ctx context.Context, p io.ReadWriter) (c *conn, err error) {
	c = new(conn)
	c.ctx = ctx
	c.in = newInput(c, p)
	c.out = p
	c.idb, err = c.id().MarshalBinary()
	if err != nil {
		return
	}

	c.eventLogger = eventLogger
	c.errorLogger = eventLogger

	return
}

func (c *conn) Read(b []byte) (int, error) {
	ch := make(chan error)
	t := 0
	go func() {
		buf := make([]byte, idsz+len(b))
		n, err := c.in.Read(buf)
		t = n - idsz
		if err != nil {
			ch <- err
			return
		}
		if n < idsz {
			ch <- errors.New("read too little")
			return
		}
		if bytes.Compare(buf[:idsz], c.idb) != 0 {
			ch <- errors.New("no header")
		}
		data := buf[idsz : idsz+t]
		n = copy(b, data)
		ch <- nil
	}()
	select {
	case <-c.ctx.Done():
		return t, ErrContextCanceled
	case err := <-ch:
		return t, err
	}
	panic("unreachable")
	return 0, nil
}

func (c *conn) Write(b []byte) (int, error) {
	ch := make(chan error)
	t := 0
	if b == nil {
		go func() {
			_, err := c.out.Write(c.idb)
			ch <- err
		}()
	} else {
		go func() {
			for t < len(b) {
				data := append(c.idb, b...)
				n, err := c.out.Write(data)
				c.eventLogger.Printf("wrote %db:\n%s", n, hex.Dump(data))
				j := n - idsz
				t += j
				if err != nil {
					ch <- err
					break
				}
			}
			ch <- nil
		}()
	}

	select {
	case <-c.ctx.Done():
		return t, ErrContextCanceled
	case err := <-ch:
		return t, err
	}
	panic("unreachable")
	return 0, nil
}

func (c *conn) Close() error {
	defer c.in.Close()
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

func (c *conn) id() uuid.UUID {
	id, ok := c.ctx.Value("id").(uuid.UUID)
	if ok {
		return id
	}
	return [idsz]byte{}
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
	return c.id().String()
}
