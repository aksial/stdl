package stdl

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
)

var eventLogger = log.New(io.Discard, "", 0)

type listener struct {
	ctx    context.Context
	cancel context.CancelFunc

	incoming chan net.Conn
	conns    map[uuid.UUID]net.Conn
	lock     sync.Mutex

	pipe io.ReadWriter

	eventLogger *log.Logger
}

func Listen(ctx context.Context, p io.ReadWriter) net.Listener {
	l := new(listener)
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.incoming = make(chan net.Conn)
	l.conns = make(map[uuid.UUID]net.Conn)
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
	fresh := make(chan net.Conn)
	// Reader loop
	go func() {
		buf := make([]byte, 65536)
		for {
			n, err := l.pipe.Read(buf)
			if err != nil {
				l.eventLogger.Printf("failed to read: %s", err)
				return
			}
			if n < idsz {
				l.eventLogger.Println("read less than header length")
				continue
			}
			l.eventLogger.Printf("read %db", n)
			id, err := uuid.FromBytes(buf[:idsz])
			if err != nil {
				l.eventLogger.Println("cannot parse header for ID")
				continue
			}
			l.lock.Lock()
			c, ok := l.conns[id]
			if !ok {
				l.eventLogger.Println("unknown connection")
				connCtx := context.WithValue(l.ctx, "id", id)
				connCtx = context.WithValue(connCtx, "disconnect", func(dcctx context.Context) {
					dcid, ok := dcctx.Value("id").(uuid.UUID)
					if ok {
						delete(l.conns, dcid)
					}
				})
				l.conns[id], err = newConn(connCtx, l.pipe)
				if err != nil {
					l.eventLogger.Printf("failed to initialize connection: %s", err)
					l.lock.Unlock()
					continue
				}
				_, err = l.conns[id].Write(nil)
				if err != nil {
					l.eventLogger.Printf("failed to write handshake to connection: %v", err)
					l.lock.Unlock()
					continue
				}
				fresh <- l.conns[id]
				l.lock.Unlock()
				continue
			}
			// Write to established connection.
			go func(conn net.Conn) {
				conn.Write(buf[idsz:n])
			}(c)
		}
	}()
	// Pass unestablished connections to the handling channel.
	for {
		select {
		case <-l.ctx.Done():
			return
		case c, ok := <-fresh:
			if !ok {
				l.eventLogger.Println("Channel for fresh connections is closed")
				return
			}
			l.eventLogger.Println("Pushing new connection to acceptor")
			l.incoming <- c
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
