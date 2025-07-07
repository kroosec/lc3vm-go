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
	OperationNOT:  (*VM).execNot,
	OperationST:   func(v *VM, inst uint16) error { return v.execStore(inst, false) },
	OperationSTI:  func(v *VM, inst uint16) error { return v.execStore(inst, true) },
	OperationSTR:  (*VM).execStoreRegister,
	OperationTRAP: (*VM).execTrap,
}
