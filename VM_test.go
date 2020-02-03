package lc3_test

import (
	"bytes"
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
				_, err := lc3.NewVM(program, nil)
				assertIsError(t, err)
			})
		}
	})

	t.Run("load an empty program and check PC", func(t *testing.T) {
		var pc uint16 = 0x3000
		program := strings.NewReader("\x30\x00")

		vm, err := lc3.NewVM(program, nil)
		assertError(t, err, nil)
		assertInitVM(t, vm, pc)
	})

	t.Run("load a simple program and check memory", func(t *testing.T) {
		var start uint16 = 0x3000
		program := strings.NewReader("\x30\x00\x12\x34")

		vm, err := lc3.NewVM(program, nil)
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, start)
		assertMemory(t, vm, start, 0x1234)
	})

	t.Run("test Simple instructions", func(t *testing.T) {
		testCases := []struct {
			name        string
			instruction string
			reg         lc3.Register
			value       uint16
			flag        uint16
		}{
			{"NOP", "\x00\x00", lc3.Register_PC, 0x3001, lc3.Flag_Z},

			// Only z flag is set at the start.
			{"BRp x3001", "\x02\x12", lc3.Register_PC, 0x3001, lc3.Flag_Z},
			{"BRz x3013", "\x04\x12", lc3.Register_PC, 0x3013, lc3.Flag_Z},
			{"BRzp x3008", "\x06\x07", lc3.Register_PC, 0x3008, lc3.Flag_Z},

			// Negative offset
			{"BRz x2f03", "\x05\x02", lc3.Register_PC, 0x2f03, lc3.Flag_Z},
			{"BRn x2f34", "\x09\x33", lc3.Register_PC, 0x3001, lc3.Flag_Z},

			{"LEA R0, x3003", "\xE0\x02", lc3.Register_R0, 0x3003, lc3.Flag_P},
			{"LEA R1, x2F35", "\xE3\x34", lc3.Register_R1, 0x2F35, lc3.Flag_P},
			{"LEA R7, x3001", "\xEE\x00", lc3.Register_R7, 0x3001, lc3.Flag_P},

			{"NOT R0, R0", "\x90\x3f", lc3.Register_R0, 0xffff, lc3.Flag_N},

			{"ADD R0, R0, R0", "\x10\x00", lc3.Register_R0, 0x0000, lc3.Flag_Z},
			{"ADD R0, R0, #0", "\x10\x20", lc3.Register_R0, 0x0000, lc3.Flag_Z},
			{"ADD R3, R2, #5", "\x16\x25", lc3.Register_R3, 0x0005, lc3.Flag_P},
			{"ADD R5, R4, #-11", "\x1B\x35", lc3.Register_R5, 0xfff5, lc3.Flag_N},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				program := strings.NewReader("\x30\x00" + test.instruction)

				vm, err := lc3.NewVM(program, nil)
				assertError(t, err, nil)

				err = vm.Step()
				assertError(t, err, nil)
				if test.reg != lc3.Register_PC {
					assertRegister(t, vm, lc3.Register_PC, 0x3001)
				}

				assertRegister(t, vm, test.reg, test.value)
				assertRegister(t, vm, lc3.Register_COND, test.flag)
			})
		}
	})

	t.Run("test PUTS trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xE0\x01\xf0\x22\x00\x41\x00\x00")

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, output)
		assertError(t, err, nil)

		// LEA R0, x3002
		err = vm.Step()
		assertError(t, err, nil)

		// PUTS (R0 == x3002 == "H\0")
		err = vm.Step()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, 0x3002)

		assertString(t, "A", output.String())
	})

	t.Run("test HALT trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x25\x00\x00")

		vm, err := lc3.NewVM(program, nil)
		assertError(t, err, nil)

		err = vm.Step()
		assertError(t, err, nil)
		assertState(t, vm.State(), lc3.StateHalted)
		assertRegister(t, vm, lc3.Register_PC, 0x3001)

		// Can't step further
		err = vm.Step()
		assertIsError(t, err)
		assertRegister(t, vm, lc3.Register_PC, 0x3001)
	})

	t.Run("execute hello-world.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/hello-world.obj")
		defer closer()

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(f, output)
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

		// LEA R0, HELLO_STR
		err = vm.Step()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, start+1)
		assertRegister(t, vm, lc3.Register_R0, 0x3003)
		assertRegister(t, vm, lc3.Register_COND, lc3.Flag_P)

		// PUTS
		err = vm.Step()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.Register_PC, start+2)
		assertString(t, "Hello World!", output.String())

		// HALT
		output.Reset()
		err = vm.Step()
		assertError(t, err, nil)
		assertState(t, vm.State(), lc3.StateHalted)
	})
}

func assertInitVM(t *testing.T, vm *lc3.VM, pc uint16) {
	t.Helper()

	assertState(t, vm.State(), lc3.StateRunning)
	for i := 0; i < lc3.MemorySize; i++ {
		value := vm.GetMemory(uint16(i))
		if value != 0 {
			t.Fatalf("Expected zeroed memory at %d, got %d", i, value)
		}
	}

	for reg := lc3.Register_R0; reg < lc3.Register_COUNT; reg++ {
		if reg == lc3.Register_PC {
			assertRegister(t, vm, reg, pc)
		} else if reg == lc3.Register_COND {
			assertRegister(t, vm, reg, lc3.Flag_Z)
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

func assertString(t *testing.T, want, got string) {
	t.Helper()

	if want != got {
		t.Fatalf("expected output to be %q, got %q", want, got)
	}
}

func assertState(t *testing.T, want uint8, got uint8) {
	t.Helper()

	if want != got {
		t.Fatalf("expected vm state to be %d, got %d", want, got)
	}
}
