package lc3

import (
	"fmt"
	"io"
	"os"
)

const (
	UserMemoryLimit = uint16(0xfdff)
	MemorySize      = int(1 << 16)
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

var registerNames map[Register]string = map[Register]string{
	Register_R0:   "R0",
	Register_R1:   "R1",
	Register_R2:   "R2",
	Register_R3:   "R3",
	Register_R4:   "R4",
	Register_R5:   "R5",
	Register_R6:   "R6",
	Register_R7:   "R7",
	Register_PC:   "PC",
	Register_COND: "COND",
}

func (reg Register) String() string {
	if reg >= Register_COUNT {
		return "INVALID"
	}
	return registerNames[reg]
}

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

var opNames map[uint8]string = map[uint8]string{
	Operation_BR:   "BR",
	Operation_ADD:  "ADD",
	Operation_LD:   "LD",
	Operation_ST:   "ST",
	Operation_JSR:  "JSR",
	Operation_AND:  "AND",
	Operation_LDR:  "LDR",
	Operation_STR:  "STR",
	Operation_RTI:  "RTI",
	Operation_NOT:  "NOT",
	Operation_LDI:  "LDI",
	Operation_STI:  "STI",
	Operation_JMP:  "JMP",
	Operation_RES:  "RES",
	Operation_LEA:  "LEA",
	Operation_TRAP: "TRAP",
}

const (
	Flag_P = uint16(1 << 0)
	Flag_Z = uint16(1 << 1)
	Flag_N = uint16(1 << 2)
)

const (
	Trap_PUTS = uint8(0x22)
	Trap_HALT = uint8(0x25)
)

const (
	StateRunning = uint8(0)
	StateHalted  = uint8(1)
)

var stateNames map[uint8]string = map[uint8]string{StateRunning: "Running", StateHalted: "Halted"}

type VM struct {
	memory    [MemorySize]uint16
	registers [Register_COUNT]uint16
	output    io.Writer
	state     uint8
}

func (v *VM) GetMemory(address uint16) uint16 {
	return v.memory[address]
}

func (v *VM) GetRegister(reg Register) uint16 {
	return v.registers[reg]
}

func NewVM(program io.Reader, output io.Writer) (*VM, error) {
	if output == nil {
		output = os.Stdout
	}
	vm := &VM{output: output, state: StateRunning}

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

func (v *VM) State() uint8 {
	return v.state
}

func (v *VM) Step() error {
	if v.state != StateRunning {
		return fmt.Errorf("VM State: %s", stateNames[v.state])
	}

	v.execInstruction()
	return nil
}

func (v *VM) execInstruction() {
	inst := v.memory[v.GetRegister(Register_PC)]
	op := uint8((inst & 0xf000) >> 12)

	switch op {
	case Operation_ADD:
		v.execAdd(inst)
	case Operation_AND:
		v.execAnd(inst)
	case Operation_BR:
		v.execBreak(inst)
	case Operation_JMP:
		v.execJump(inst)
	case Operation_JSR:
		v.execJumpSubroutine(inst)
	case Operation_LD:
		v.execLoad(inst, false)
	case Operation_LDI:
		v.execLoad(inst, true)
	case Operation_LDR:
		v.execLoadRegister(inst)
	case Operation_LEA:
		v.execLoadEffectiveAddress(inst)
	case Operation_NOT:
		v.execNot(inst)
	case Operation_ST:
		v.execStore(inst, false)
	case Operation_STI:
		v.execStore(inst, true)
	case Operation_RTI, Operation_RES:
		panic(fmt.Sprintf("Operation %q not implemented", opNames[op]))
	case Operation_TRAP:
		v.execTrap(inst)
	}

	if doIncrementPC(op) {
		v.incrementRegister(Register_PC, 1)
	}
}

func doIncrementPC(op uint8) bool {
	return op != Operation_JMP && op != Operation_JSR
}

func (v *VM) updateFlags(reg Register) {
	value := v.GetRegister(reg)

	flags := Flag_P
	if value == 0 {
		flags = Flag_Z
	} else if value>>15 == 1 {
		flags = Flag_N
	}
	v.setRegister(Register_COND, flags)
}

func (v *VM) setRegister(reg Register, value uint16) {
	v.registers[reg] = value
}

func (v *VM) execAdd(inst uint16) {
	destination := Register((inst >> 9) & 0x7)
	source1 := Register((inst >> 6) & 0x7)

	var value uint16
	if inst&0x32 == 0 {
		source2 := Register(inst & 0x7)

		value = v.GetRegister(source2)
	} else {
		value = signExtend(inst, 5)
	}

	v.setRegister(destination, v.GetRegister(source1)+value)
	v.updateFlags(destination)
}

func (v *VM) execAnd(inst uint16) {
	destination := Register((inst >> 9) & 0x7)
	source1 := Register((inst >> 6) & 0x7)

	var value uint16
	if inst&0x32 == 0 {
		source2 := Register(inst & 0x7)

		value = v.GetRegister(source2)
	} else {
		value = signExtend(inst, 5)
	}

	v.setRegister(destination, v.GetRegister(source1)&value)
	v.updateFlags(destination)
}

func (v *VM) execNot(inst uint16) {
	// XXX: Check trailing 1's ?
	destination := Register((inst >> 9) & 0x7)
	source := Register((inst >> 6) & 0x7)
	value := v.GetRegister(source) | 0xffff

	v.setRegister(destination, value)
	v.updateFlags(destination)
}

func (v *VM) execLoad(inst uint16, indirect bool) {
	destination := Register((inst >> 9) & 0x7)
	offset := signExtend(inst, 8)
	value := v.GetMemory(v.GetRegister(Register_PC) + offset + 1)
	if indirect {
		value = v.GetMemory(value)
	}

	v.setRegister(destination, value)
	v.updateFlags(destination)
}

func (v *VM) execStore(inst uint16, indirect bool) {
	source := Register((inst >> 9) & 0x7)
	offset := signExtend(inst, 9)
	address := v.GetRegister(Register_PC) + offset + 1
	if indirect {
		address = v.GetMemory(address)
	}

	v.setMemory(address, v.GetRegister(source))
}

func (v *VM) setMemory(address uint16, value uint16) {
	v.memory[address] = value
}

func (v *VM) execLoadRegister(inst uint16) {
	destination := Register((inst >> 9) & 0x7)
	base := Register((inst >> 6) & 0x7)
	offset := signExtend(inst, 6)
	value := v.GetMemory(v.GetRegister(base) + offset)

	v.setRegister(destination, value)
	v.updateFlags(destination)
}

func (v *VM) execTrap(inst uint16) {
	trap := uint8(inst & 0x00ff)

	switch trap {
	case Trap_PUTS:
		v.trapPuts()
	case Trap_HALT:
		v.trapHalt()
	default:
		panic(fmt.Sprintf("Trap 0x%x not implemented", trap))
	}
}

func (v *VM) trapHalt() {
	v.state = StateHalted
}

func (v *VM) trapPuts() {
	address := v.GetRegister(Register_R0)

	var out []byte
	for {
		// XXX: Validate that value is less or equal to 0xff too.
		value := v.GetMemory(address)
		if value == 0 {
			break
		}

		out = append(out, byte(value))
		if address == UserMemoryLimit {
			break
		}
		address++
	}

	v.output.Write(out)
}

func (v *VM) execLoadEffectiveAddress(inst uint16) {
	offset := signExtend(inst, 9)
	reg := Register((inst >> 9) & 0x7)

	v.incrementRegister(reg, v.GetRegister(Register_PC)+offset+1)
	v.updateFlags(reg)
}

func (v *VM) execBreak(inst uint16) {
	offset := signExtend(inst, 9)
	flags := (inst >> 9) & 0x7

	if v.registers[Register_COND]&flags != 0 {
		v.incrementRegister(Register_PC, offset)
	}
}

func (v *VM) execJump(inst uint16) {
	baseRegister := Register((inst >> 6) & 0x7)

	v.setRegister(Register_PC, v.GetRegister(baseRegister))
}

func (v *VM) execJumpSubroutine(inst uint16) {
	v.setRegister(Register_R7, v.GetRegister(Register_PC)+1)

	var destination uint16
	if inst&0x800 == 0 {
		baseRegister := Register((inst >> 6) & 0x7)
		destination = v.GetRegister(baseRegister)
	} else {
		destination = signExtend(inst, 11)
	}

	v.setRegister(Register_PC, destination)
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

		v.setMemory(address, value)
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

func signExtend(value uint16, count uint8) uint16 {
	value = value & ((1 << count) - 1)
	if (value>>(count-1))&1 != 0 {
		value |= (0xFFFF << count)
	}
	return value
}
