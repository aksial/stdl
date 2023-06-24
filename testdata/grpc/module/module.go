package main

import (
	"encoding/binary"
	"os"
)

func main() {
	var d uint32
	err := binary.Read(os.Stdin, binary.LittleEndian, &d)
	if err != nil {
		panic(err)
	}
	err = binary.Write(os.Stdout, binary.LittleEndian, d*2)
	if err != nil {
		panic(err)
	}
}
