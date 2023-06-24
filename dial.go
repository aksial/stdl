package stdl

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
)

var ErrContextCanceled error = errors.New("context is done")

func Dial(ctx context.Context, p io.ReadWriter, opts ...DialOption) (net.Conn, error) {
	// Create connection.
	c, err := newConn(ctx, p)
	if err != nil {
		return nil, err
	}

	// Apply DialOptions.
	for _, opt := range opts {
		if err := opt.apply(c); err != nil {
			return nil, err
		}
	}

	return c, err
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
