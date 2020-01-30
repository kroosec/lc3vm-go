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

const (
	Operation_BR = iota
	Operation_ADD
	Operation_LD
	Operation_ST
	Operation_JSR
	Operation_AND
	Operation_LDR
	Operation_STR
	Operation_RTI
	Operation_NOT
	Operation_LDI
	Operation_STI
	Operation_JMP
	Operation_RES
	Operation_LEA
	Operation_TRAP
)

const (
	Flag_P = uint16(1 << 0)
	Flag_Z = uint16(1 << 1)
	Flag_N = uint16(1 << 2)
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

	vm.registers[Register_COND] = Flag_Z

	return vm, nil
}

func (v *VM) Step() error {
	if err := v.execInstruction(); err != nil {
		// XXX: Do not increment on error ?
		return err
	}

	v.incrementRegister(Register_PC, 1)
	return nil
}

func (v *VM) execInstruction() (err error) {
	inst := v.memory[v.GetRegister(Register_PC)]
	op := uint8(inst & 0xf000)

	switch op {
	case Operation_BR:
		err = v.execBreak(inst)
	default:
		err = fmt.Errorf("Operation %x not implemented", op)
	}

	return err
}

func (v *VM) execBreak(inst uint16) error {
	offset := signExtend(inst&0x1ff, 9)
	flags := (inst >> 9) & 0x7

	if v.registers[Register_COND]&flags != 0 {
		v.incrementRegister(Register_PC, offset)
	}

	return nil
}

func (v *VM) incrementRegister(reg Register, value uint16) {
	v.registers[reg] += value
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

func signExtend(value uint16, pos uint8) uint16 {
	if (value>>(pos-1))&1 != 0 {
		value |= (0xFFFF << pos)
	}
	return value
}
