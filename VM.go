package lc3

import (
	"fmt"
	"io"
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

func (v *VM) GetMemory(address uint16) uint16 {
	return v.memory[address]
}

func (v *VM) GetRegister(reg Register) uint16 {
	return v.registers[reg]
}

func NewVM(program io.Reader) (*VM, error) {
	vm := &VM{}

	// .ORIG / Start address.
	if err := vm.readStart(program); err != nil {
		return nil, err
	}

	if err := vm.readProgram(program); err != nil {
		return nil, err
	}

	return vm, nil
}

func (v *VM) readProgram(program io.Reader) error {
	address := v.GetRegister(Register_PC)

	for {
		value, err := readValue(program)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("Error reading the program: %v", err)
		}

		v.memory[address] = value
		address++
		if address == 0 {
			// XXX: Any further restrictions on programs size ?
			return fmt.Errorf("Program size beyond memory space.")
		}
	}
}

func (v *VM) readStart(program io.Reader) error {
	// XXX: Any restrictions on Program Counter value ?
	pc, err := readValue(program)
	if err != nil {
		return fmt.Errorf("Failed to read orig value from program: %v", err)
	}

	v.registers[Register_PC] = pc
	return nil
}

func readValue(program io.Reader) (uint16, error) {
	var buffer [2]byte

	n, err := program.Read(buffer[:])
	if err != nil {
		return 0, err
	}
	if n != 2 {
		return 0, fmt.Errorf("Expected 2 bytes, got 1")
	}

	return (uint16(buffer[0]) << 8) + uint16(buffer[1]), nil
}
