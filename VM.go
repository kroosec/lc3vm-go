package lc3

import (
	"fmt"
	"io"
	"io/ioutil"
)

const (
	MemorySize = int(1 << 16)
)

type Register uint8

const (
	Register_R0 Register = iota
	Register_R1
	Register_R2
	Register_R3
	Register_R4
	Register_R5
	Register_R6
	Register_R7
	Register_PC
	Register_COND

	Register_COUNT
)

type VM struct {
	memory    [MemorySize]uint16
	registers [Register_COUNT]uint16
}

func NewVM() *VM {
	return &VM{}
}

func (v *VM) GetMemory(address uint16) uint16 {
	return v.memory[address]
}

func (v *VM) GetRegister(reg Register) uint16 {
	return v.registers[reg]
}

func (v *VM) Load(program io.Reader) error {
	content, err := ioutil.ReadAll(program)
	if err != nil {
		return fmt.Errorf("Failed to load program: %v", err)
	}

	// Program must contain at least .ORIG, and must be multiple of 2, as each instruction is 16 bits long.
	if len(content) < 2 || len(content)%2 != 0 {
		return fmt.Errorf("Program size in bytes not multiple of two.")
	}

	// XXX: Get first 2 bytes: Set the PC.

	for i := 2; i < len(content); i += 2 {
		// XXX: For each two bytes, load as instruction (swap16?) into memory.
	}

	return nil
}
