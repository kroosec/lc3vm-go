package lc3_test

import (
	"fmt"
	"strings"
	"testing"

	"lc3"
)

func TestVM(t *testing.T) {
	t.Run("create a new VM and check zeroed memory", func(t *testing.T) {
		vm := lc3.NewVM()
		assertInitVM(t, vm)
	})

	t.Run("try to load erroneous programs", func(t *testing.T) {
		vm := lc3.NewVM()
		assertInitVM(t, vm)

		testCases := []string{
			"",
			"\x00",
			"\x01\x02\x03",
		}

		for i, test := range testCases {
			t.Run(fmt.Sprintf("test case #%d", i), func(t *testing.T) {
				program := strings.NewReader(test)
				err := vm.Load(program)
				assertIsError(t, err)
			})
		}
	})

	t.Run("load a simple program and check PC", func(t *testing.T) {
		vm := lc3.NewVM()
		assertInitVM(t, vm)

		program := strings.NewReader("\x00\x30")
		err := vm.Load(program)
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, 0x3000)
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
		assertRegister(t, vm, reg, 0)
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if got != want {
		t.Fatalf("expected error %+v, got %+v", want, got)
	}
}

func assertIsError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error, got none")
	}
}

func assertRegister(t *testing.T, vm *lc3.VM, reg lc3.Register, want uint16) {
	t.Helper()

	got := vm.GetRegister(reg)
	if got != want {
		t.Fatalf("expected register %d's value to be 0x%x, got 0x%x", reg, want, got)
	}
}
