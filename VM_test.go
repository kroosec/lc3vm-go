package lc3_test

import (
	"testing"

	"lc3"
)

func TestVM(t *testing.T) {
	t.Run("create a new VM and check zeroed memory", func(t *testing.T) {
		vm := lc3.NewVM()
		assertInitVM(t, vm)
	})
}

func assertInitVM(t *testing.T, vm *lc3.VM) {
	t.Helper()

	for i := 0; i < lc3.MemorySize; i++ {
		value := vm.GetMemory(uint16(i))
		if value != 0 {
			t.Fatalf("Expected zeroed memory at %d, got %d", i, value)
		}
	}

	for reg := lc3.Register_R0; reg < lc3.Register_COUNT; reg++ {
		value := vm.GetRegister(reg)
		if value != 0 {
			t.Fatalf("expected register %d's value to be zero, got %d", reg, value)
		}
	}
}
