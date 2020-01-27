package lc3

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
