package stdl

import (
	"context"
	_ "embed"
	"encoding/binary"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed testdata/grpc/module/module.wasm
var moduleData []byte

func TestWASM(t *testing.T) {
	// Prepare context and runtime.
	ctx, cancel := context.WithCancel(context.Background())
	rt := wazero.NewRuntime(ctx)
	defer cancel()
	defer rt.Close(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	// Run the module's main (_start) function with custom stdin & stdout.
	p := Pipe()
	go func() {
		if _, err := rt.InstantiateWithConfig(ctx, moduleData, wazero.NewModuleConfig().WithStdin(p).WithStdout(p)); err != nil {
			t.Log(err)
		}
	}()

	// Write data to the module's stdin.
	var data uint32 = 123
	err := binary.Write(p, binary.LittleEndian, data)
	if err != nil {
		t.Fatal(err)
	}

	// We've written to stdin. The module will write the doubled value back
	// to its stdout, which is the pipe we created earlier.
	var d uint32
	err = binary.Read(p, binary.LittleEndian, &d)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Sent u%d, got back u%d (%d * 2 = %d)", data, d, data, data*2)
	if data*2 != d {
		t.Error("Input and output don't match")
	}
}
