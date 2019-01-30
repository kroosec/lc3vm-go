package lc3_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"lc3"
)

func TestVM(t *testing.T) {
	t.Run("try to load erroneous programs", func(t *testing.T) {
		testCases := []string{
			"",
			"\x30",
			"\x30\x00\x08",
		}

		for i, test := range testCases {
			t.Run(fmt.Sprintf("test case #%d", i), func(t *testing.T) {
				program := strings.NewReader(test)
				_, err := lc3.NewVM(program)
				assertIsError(t, err)
			})
		}
	})

	t.Run("load a empty program and check PC", func(t *testing.T) {
		var pc uint16 = 0x3000
		program := strings.NewReader("\x30\x00")

		vm, err := lc3.NewVM(program)
		assertError(t, err, nil)
		assertInitVM(t, vm, pc)
	})

	t.Run("load a simple program and check memory", func(t *testing.T) {
		var start uint16 = 0x3000
		program := strings.NewReader("\x30\x00\x12\x34")

		vm, err := lc3.NewVM(program)
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, start)
		assertMemory(t, vm, start, 0x1234)
	})

	t.Run("load a hello-world.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/hello-world.obj")
		defer closer()

		vm, err := lc3.NewVM(f)
		assertError(t, err, nil)

		var start uint16 = 0x3000
		values := []uint16{0xe002, 0xf022, 0xf025, 0x0048,
			0x0065, 0x006c, 0x006c, 0x006f,
			0x0020, 0x0057, 0x006f, 0x0072,
			0x006c, 0x0064, 0x0021, 0x0000}

		assertRegister(t, vm, lc3.Register_PC, start)
		for i, value := range values {
			assertMemory(t, vm, start+uint16(i), value)
		}
	})
}

func assertInitVM(t *testing.T, vm *lc3.VM, pc uint16) {
	t.Helper()

	for i := 0; i < lc3.MemorySize; i++ {
		value := vm.GetMemory(uint16(i))
		if value != 0 {
			t.Fatalf("Expected zeroed memory at %d, got %d", i, value)
		}
	}

	for reg := lc3.Register_R0; reg < lc3.Register_COUNT; reg++ {
		if reg == lc3.Register_PC {
			assertRegister(t, vm, reg, pc)
		} else {
			assertRegister(t, vm, reg, 0)
		}
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

func assertMemory(t *testing.T, vm *lc3.VM, address, want uint16) {
	t.Helper()

	got := vm.GetMemory(address)
	if got != want {
		t.Fatalf("expected memory at %x's to have 0x%x, got 0x%x", address, want, got)
	}
}

func openTestfile(t *testing.T, name string) (*os.File, func() error) {
	t.Helper()

	f, err := os.Open(name)
	assertError(t, err, nil)

	return f, f.Close
}
