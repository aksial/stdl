package stdl

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	p := Pipe()
	pass := make(chan error)
	data := []byte("hello")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	//SetLogger(log.New(os.Stdout, "[Default] ", log.Lmicroseconds|log.Lshortfile))
	il := Listen(ctx, p)

	// Server
	go func(ctx context.Context, l net.Listener) {
		ctx, cancel := context.WithCancelCause(ctx)
		accept := make(chan net.Conn)
		go func() {
			for {
				//t.Log("[T] Ready to accept")
				c, err := l.Accept()
				if err != nil {
					//t.Logf("[T] Failed to accept: %s", err)
					pass <- err
					cancel(err)
					break
				}
				//t.Log("[T] Pushing accepted connection")
				accept <- c
			}
		}()
		for {
			select {
			case <-ctx.Done():
				break
			case c, ok := <-accept:
				if !ok {
					continue
				}
				if c == nil {
					continue
				}
				t.Logf("Accepted connection from %v", c.RemoteAddr())
				go func(c net.Conn) {
					if c == nil {
						pass <- fmt.Errorf("got nil net.Conn in goroutine")
						return
					}
					b := make([]byte, len(data))
					t.Logf("Reading...")
					n, err := c.Read(b)
					if err != nil {
						pass <- err
						return
					}
					t.Logf("Received %d bytes: %v", n, b[:n])
					if bytes.Compare(b[:n], data) == 0 {
						pass <- nil
					} else {
						pass <- fmt.Errorf("unexpected comparison result")
					}
				}(c)
			}
		}
	}(ctx, il)

	// Client
	time.Sleep(time.Second)
	c, err := Dial(
		ctx, p,
		WithErrorLogger(log.New(os.Stdout, "[Client Error] ", log.Lmicroseconds|log.Lshortfile)),
		//WithEventLogger(log.New(os.Stdout, "[Client Event] ", log.Lmicroseconds|log.Lshortfile)),
	)
	if err != nil {
		t.Fatal(err)
	}
	n, err := c.Write(data)
	if n < len(data) {
		t.Logf("Only wrote %db", n)
	}
	if err != nil {
		t.Fatal(err)
	}
	timeout := time.NewTimer(time.Second * 5)
	select {
	case <-timeout.C:
		t.Error("did not receive expected data in time")
	case err, ok := <-pass:
		if !ok {
			t.Log("pass channel closed")
		}
		if err != nil {
			t.Fatalf("unexpected data received: %s", err)
		} else {
			t.Log("expected data received")
		}
	}
}
