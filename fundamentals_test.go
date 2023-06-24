package stdl

import (
	"testing"
)

func TestFundamentals(t *testing.T) {
	// Prepare a simple io.ReadWriter
	p := Pipe()

	// Create a goroutine that will read once from the pipe, and then
	// put the number of written bytes to the result channel.
	res := make(chan int)
	go func() {
		buf := make([]byte, 256)
		n, err := p.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		res <- n
	}()

	// Write some data to the pipe and take note of the bytes written.
	data := []byte("hello")
	n, err := p.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(data) {
		t.Fatalf("Only wrote %d bytes, must be %d", n, len(data))
	}
	// Wait for a message on the result channel. Once we get back the number of
	// bytes the goroutine we created earlier read from the pipe, compare to the
	// number of bytes we wrote to the pipe.
	m := <-res
	if m != n {
		t.Fail()
	}
	t.Logf("Sent %db, got back %db", n, m)
}
