package stdl

import (
	"context"
	"errors"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
)

const idsz int = 16

var ErrContextCanceled error = errors.New("context is done")

func Dial(ctx context.Context, p io.ReadWriter, opts ...DialOption) (net.Conn, error) {
	// Return on timeout, error, or server hello.

	handshakeError := make(chan error)

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	connCtx := context.WithValue(ctx, "id", id)
	c, err := newConn(connCtx, p)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt.apply(c); err != nil {
			return nil, err
		}
	}

	// Start handshake
	go func() {
		n, err := c.Write(nil)
		if err != nil {
			c.errorLogger.Printf("handshake write error: %s", err)
			handshakeError <- err
			return
		}
		buf := make([]byte, idsz)
		n, err = c.Read(buf)
		if err != nil {
			c.errorLogger.Printf("handshake read error: %s", err)
			handshakeError <- err
			return
		}
		if n != 0 {
			err = errors.New(string(buf[:n]))
			c.errorLogger.Printf("failed to establish connection: %s", err)
			handshakeError <- err
			return
		}
		handshakeError <- nil
	}()

	for {
		select {
		case <-ctx.Done():
			if err := context.Cause(ctx); err != nil {
				return nil, err
			}
			return nil, ErrContextCanceled
		case err := <-handshakeError:
			if err != nil {
				c.errorLogger.Print(err)
				c.Close()
			} else {
				c.eventLogger.Print("connection established")
			}
			return c, err
		default:
		}
	}

	panic("unreachable")
	return nil, nil
}

type DialOption interface {
	apply(*conn) error
}

type dialOptionEventLogger log.Logger

func (opt *dialOptionEventLogger) apply(c *conn) error {
	c.setEventLogger((*log.Logger)(opt))
	return nil
}

func WithEventLogger(logger *log.Logger) DialOption {
	return (*dialOptionEventLogger)(logger)
}

type dialOptionErrorLogger log.Logger

func (opt *dialOptionErrorLogger) apply(c *conn) error {
	c.setErrorLogger((*log.Logger)(opt))
	return nil
}

func WithErrorLogger(logger *log.Logger) DialOption {
	return (*dialOptionErrorLogger)(logger)
}
