package lc3

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const (
	UserMemoryLimit = uint16(0xfdff)
	MemorySize      = int(1 << 16)

	MemoryKBSR = uint16(0xFE00)
	MemoryKBDR = uint16(0xFE02)
)

type Register uint8

const (
	RegisterR0 Register = iota
	RegisterR1
	RegisterR2
	RegisterR3
	RegisterR4
	RegisterR5
	RegisterR6
	RegisterR7
	RegisterPC
	RegisterCOND

	RegisterCOUNT
)

var registerNames map[Register]string = map[Register]string{
	RegisterR0:   "R0",
	RegisterR1:   "R1",
	RegisterR2:   "R2",
	RegisterR3:   "R3",
	RegisterR4:   "R4",
	RegisterR5:   "R5",
	RegisterR6:   "R6",
	RegisterR7:   "R7",
	RegisterPC:   "PC",
	RegisterCOND: "COND",
}

func (reg Register) String() string {
	if reg >= RegisterCOUNT {
		return "INVALID"
	}
	return registerNames[reg]
}

const (
	OperationBR = iota
	OperationADD
	OperationLD
	OperationST
	OperationJSR
	OperationAND
	OperationLDR
	OperationSTR
	OperationRTI
	OperationNOT
	OperationLDI
	OperationSTI
	OperationJMP
	OperationRES
	OperationLEA
	OperationTRAP
)

var opNames map[uint8]string = map[uint8]string{
	OperationBR:   "BR",
	OperationADD:  "ADD",
	OperationLD:   "LD",
	OperationST:   "ST",
	OperationJSR:  "JSR",
	OperationAND:  "AND",
	OperationLDR:  "LDR",
	OperationSTR:  "STR",
	OperationRTI:  "RTI",
	OperationNOT:  "NOT",
	OperationLDI:  "LDI",
	OperationSTI:  "STI",
	OperationJMP:  "JMP",
	OperationRES:  "RES",
	OperationLEA:  "LEA",
	OperationTRAP: "TRAP",
}

const (
	FlagP = uint16(1 << 0)
	FlagZ = uint16(1 << 1)
	FlagN = uint16(1 << 2)
)

const (
	TrapGETC = uint8(0x20)
	TrapOUT  = uint8(0x21)
	TrapPUTS = uint8(0x22)
	TrapHALT = uint8(0x25)
)

const (
	StateRunning = uint8(0)
	StateHalted  = uint8(1)
)

var stateNames map[uint8]string = map[uint8]string{StateRunning: "Running", StateHalted: "Halted"}

type VM struct {
	memory    [MemorySize]uint16
	registers [RegisterCOUNT]uint16
	output    io.Writer
	input     *bufio.Reader
	state     uint8
}

func (v *VM) GetMemory(address uint16) (uint16, error) {
	if address == MemoryKBSR {
		v.memory[MemoryKBSR] = 0
		if v.peekChar() {
			char, err := v.getChar()
			if err != nil {
				return 0, fmt.Errorf("peeked char, but couldn't read it: %v", err)
			}

			v.memory[MemoryKBSR] = (1 << 15)
			v.memory[MemoryKBDR] = uint16(char)
		}
	}

	return v.memory[address], nil
}

func (v *VM) GetRegister(reg Register) uint16 {
	return v.registers[reg]
}

func NewVM(program io.Reader, input io.Reader, output io.Writer) (*VM, error) {
	if output == nil {
		output = os.Stdout
	}
	if input == nil {
		input = os.Stdin
	}
	vm := &VM{input: bufio.NewReader(input), output: output, state: StateRunning}

	// .ORIG / Start address.
	if err := vm.readStart(program); err != nil {
		return nil, err
	}

	if err := vm.readProgram(program); err != nil {
		return nil, err
	}

	vm.registers[RegisterCOND] = FlagZ

	return vm, nil
}

func (v *VM) State() uint8 {
	return v.state
}

func (v *VM) Step() error {
	if v.state != StateRunning {
		return fmt.Errorf("VM State: %s", stateNames[v.state])
	}

	return v.execInstruction()
}

func (v *VM) Run() error {
	for v.State() == StateRunning {
		if err := v.Step(); err != nil {
			return err
		}
	}
	return nil
}

func (v *VM) execInstruction() error {
	inst, err := v.GetMemory(v.GetRegister(RegisterPC))
	if err != nil {
		return err
	}
	op := uint8((inst & 0xf000) >> 12)

	if exec, ok := instructions[op]; ok {
		if err := exec(v, inst); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Operation %q not implemented", opNames[op])
	}

	if doIncrementPC(op) {
		v.incrementRegister(RegisterPC, 1)
	}
	return nil
}

func doIncrementPC(op uint8) bool {
	return op != OperationJMP && op != OperationJSR
}

func (v *VM) updateFlags(reg Register) {
	value := v.GetRegister(reg)

	flags := FlagP
	if value == 0 {
		flags = FlagZ
	} else if value>>15 == 1 {
		flags = FlagN
	}
	v.SetRegister(RegisterCOND, flags)
}

func (v *VM) SetRegister(reg Register, value uint16) {
	v.registers[reg] = value
}

func (v *VM) execAnd(inst uint16) {
	destination := Register((inst >> 9) & 0x7)
	source1 := Register((inst >> 6) & 0x7)

	var value uint16
	if inst&0x0020 == 0 {
		source2 := Register(inst & 0x7)

		value = v.GetRegister(source2)
	} else {
		value = signExtend(inst, 5)
	}

	v.SetRegister(destination, v.GetRegister(source1)&value)
	v.updateFlags(destination)
}

func (v *VM) execNot(inst uint16) {
	// XXX: Check trailing 1's ?
	destination := Register((inst >> 9) & 0x7)
	source := Register((inst >> 6) & 0x7)
	value := v.GetRegister(source) ^ 0xffff

	v.SetRegister(destination, value)
	v.updateFlags(destination)
}

func (v *VM) execLoad(inst uint16, indirect bool) error {
	destination := Register((inst >> 9) & 0x7)
	offset := signExtend(inst, 9)
	value, err := v.GetMemory(v.GetRegister(RegisterPC) + offset + 1)
	if err != nil {
		return err
	}
	if indirect {
		value, err = v.GetMemory(value)
		if err != nil {
			return err
		}
	}

	v.SetRegister(destination, value)
	v.updateFlags(destination)
	return nil
}

func (v *VM) execStore(inst uint16, indirect bool) error {
	source := Register((inst >> 9) & 0x7)
	offset := signExtend(inst, 9)
	address := v.GetRegister(RegisterPC) + offset + 1
	if indirect {
		var err error
		address, err = v.GetMemory(address)
		if err != nil {
			return err
		}
	}

	v.SetMemory(address, v.GetRegister(source))
	return nil
}

func (v *VM) execStoreRegister(inst uint16) error {
	source := Register((inst >> 9) & 0x7)
	base := Register((inst >> 6) & 0x7)
	offset := signExtend(inst, 6)
	address := v.GetRegister(base) + offset

	v.SetMemory(address, v.GetRegister(source))
	return nil
}

func (v *VM) SetMemory(address uint16, value uint16) {
	v.memory[address] = value
}

func (v *VM) execLoadRegister(inst uint16) error {
	destination := Register((inst >> 9) & 0x7)
	base := Register((inst >> 6) & 0x7)
	offset := signExtend(inst, 6)
	value, err := v.GetMemory(v.GetRegister(base) + offset)
	if err != nil {
		return err
	}

	v.SetRegister(destination, value)
	v.updateFlags(destination)
	return nil
}

func (v *VM) execTrap(inst uint16) error {
	trap := uint8(inst & 0x00ff)

	switch trap {
	case TrapGETC:
		if err := v.trapGetc(); err != nil {
			return err
		}
	case TrapOUT:
		if err := v.trapOut(); err != nil {
			return err
		}
	case TrapPUTS:
		if err := v.trapPuts(); err != nil {
			return err
		}
	case TrapHALT:
		v.trapHalt()
	default:
		return fmt.Errorf("trap 0x%x not implemented", trap)
	}
	return nil
}

func (v *VM) execLoadEffectiveAddress(inst uint16) {
	offset := signExtend(inst, 9)
	reg := Register((inst >> 9) & 0x7)

	v.SetRegister(reg, v.GetRegister(RegisterPC)+offset+1)
	v.updateFlags(reg)
}

func (v *VM) execBreak(inst uint16) {
	offset := signExtend(inst, 9)
	flags := (inst >> 9) & 0x7

	if v.registers[RegisterCOND]&flags != 0 {
		v.incrementRegister(RegisterPC, offset)
	}
}

func (v *VM) execJump(inst uint16) {
	baseRegister := Register((inst >> 6) & 0x7)

	v.SetRegister(RegisterPC, v.GetRegister(baseRegister))
}

func (v *VM) execJumpSubroutine(inst uint16) {
	v.SetRegister(RegisterR7, v.GetRegister(RegisterPC)+1)

	var destination uint16
	if inst&0x800 == 0 {
		baseRegister := Register((inst >> 6) & 0x7)
		destination = v.GetRegister(baseRegister)
	} else {
		destination = v.GetRegister(RegisterPC) + signExtend(inst, 11) + 1
	}

	v.SetRegister(RegisterPC, destination)
}

func (v *VM) incrementRegister(reg Register, value uint16) {
	v.registers[reg] += value
}

func (v *VM) readProgram(program io.Reader) error {
	address := v.GetRegister(RegisterPC)

	for {
		value, err := readValue(program)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("Error reading the program: %v", err)
		}

		v.SetMemory(address, value)
		if address == UserMemoryLimit {
			return nil
		}
		address++
	}
}

func (v *VM) readStart(program io.Reader) error {
	// XXX: Any restrictions on Program Counter value ?
	pc, err := readValue(program)
	if err != nil {
		return fmt.Errorf("Failed to read orig value from program: %v", err)
	}

	v.registers[RegisterPC] = pc
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

func signExtend(value uint16, count uint8) uint16 {
	value = value & ((1 << count) - 1)
	if (value>>(count-1))&1 != 0 {
		value |= (0xFFFF << count)
	}
	return value
}
