package lc3

type instruction func(v *VM, inst uint16) error

var instructions = map[uint8]instruction{
	OperationADD:  func(v *VM, inst uint16) error { v.execAdd(inst); return nil },
	OperationAND:  func(v *VM, inst uint16) error { v.execAnd(inst); return nil },
	OperationBR:   func(v *VM, inst uint16) error { v.execBreak(inst); return nil },
	OperationJMP:  func(v *VM, inst uint16) error { v.execJump(inst); return nil },
	OperationJSR:  func(v *VM, inst uint16) error { v.execJumpSubroutine(inst); return nil },
	OperationLD:   func(v *VM, inst uint16) error { return v.execLoad(inst, false) },
	OperationLDI:  func(v *VM, inst uint16) error { return v.execLoad(inst, true) },
	OperationLDR:  (*VM).execLoadRegister,
	OperationLEA:  func(v *VM, inst uint16) error { v.execLoadEffectiveAddress(inst); return nil },
	OperationNOT:  func(v *VM, inst uint16) error { v.execNot(inst); return nil },
	OperationST:   func(v *VM, inst uint16) error { return v.execStore(inst, false) },
	OperationSTI:  func(v *VM, inst uint16) error { return v.execStore(inst, true) },
	OperationSTR:  (*VM).execStoreRegister,
	OperationTRAP: (*VM).execTrap,
}

func (v *VM) execAdd(inst uint16) {
	destination := Register((inst >> 9) & 0x7)
	source1 := Register((inst >> 6) & 0x7)

	var value uint16
	if inst&0x0020 == 0 {
		source2 := Register(inst & 0x7)

		value = v.GetRegister(source2)
	} else {
		value = signExtend(inst, 5)
	}

	v.SetRegister(destination, v.GetRegister(source1)+value)
	v.updateFlags(destination)
}

