package stdl

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
)

var eventLogger = log.New(io.Discard, "", 0)

type listener struct {
	ctx    context.Context
	cancel context.CancelFunc

	incoming chan net.Conn
	conn     *conn

	pipe io.ReadWriter

	eventLogger *log.Logger
}

func Listen(ctx context.Context, p io.ReadWriter) net.Listener {
	l := new(listener)
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.incoming = make(chan net.Conn)
	l.pipe = p
	l.eventLogger = eventLogger
	go l.do()
	return l
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case <-l.ctx.Done():
		if err := context.Cause(l.ctx); err != nil {
			return nil, err
		}
		return nil, ErrContextCanceled
	case c, ok := <-l.incoming:
		if !ok {
			return nil, errors.New("channel is closed")
		}
		l.eventLogger.Println("Accepted connection")
		return c, nil
	}
}

func (l *listener) do() {
	defer l.Close()
	// Reader loop
	go func() {
		buf := make([]byte, 65536)
		for {
			n, err := l.pipe.Read(buf)
			if err != nil {
				l.eventLogger.Printf("failed to read: %s", err)
				return
			}
			l.eventLogger.Printf("read %db", n)
			if l.conn == nil {
				connCtx := context.WithValue(l.ctx, "disconnect", func(_ context.Context) {
					l.conn = nil
				})
				l.conn, err = newConn(connCtx, l.pipe)
				if err != nil {
					l.eventLogger.Printf("failed to initialize connection: %s", err)
					l.conn = nil
					continue
				}
				l.incoming <- l.conn
			}
			l.conn.Write(buf[:n])
		}
	}()
	// Pass unestablished connections to the handling channel.
	for {
		select {
		case <-l.ctx.Done():
			return
		}
	}
}

func (l *listener) Close() error {
	defer l.cancel()
	return nil
}

func (l *listener) Addr() net.Addr {
	return l
}

func (l *listener) Network() string {
	return "io"
}

func (l *listener) String() string {
	return "io"
}

func SetLogger(logger *log.Logger) {
	eventLogger = logger
}
