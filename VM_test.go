package lc3_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	lc3 "github.com/kroosec/lc3vm-go"
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
				_, err := lc3.NewVM(program, nil, nil)
				assertIsError(t, err)
			})
		}
	})

	t.Run("load an empty program and check PC", func(t *testing.T) {
		var pc uint16 = 0x3000
		program := strings.NewReader("\x30\x00")

		vm, err := lc3.NewVM(program, nil, nil)
		assertError(t, err, nil)
		assertInitVM(t, vm, pc)
	})

	t.Run("load a simple program and check memory", func(t *testing.T) {
		var start uint16 = 0x3000
		program := strings.NewReader("\x30\x00\x12\x34")

		vm, err := lc3.NewVM(program, nil, nil)
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.RegisterPC, start)
		assertMemory(t, vm, start, 0x1234)
	})

	t.Run("test Simple instructions", func(t *testing.T) {
		var canaryAddress, canaryValue uint16 = 0x2f06, 0x1234

		testCases := []struct {
			name        string
			instruction string
			steps       int
			reg         lc3.Register
			value       uint16
			flag        uint16
		}{
			{"NOP", "\x00\x00", 1, lc3.RegisterPC, 0x3001, lc3.FlagZ},

			{"BRp x3001", "\x02\x12", 1, lc3.RegisterPC, 0x3001, lc3.FlagZ},
			{"BRz x3013", "\x04\x12", 1, lc3.RegisterPC, 0x3013, lc3.FlagZ},
			{"BRzp x3008", "\x06\x07", 1, lc3.RegisterPC, 0x3008, lc3.FlagZ},

			{"BRz x2f03", "\x05\x02", 1, lc3.RegisterPC, 0x2f03, lc3.FlagZ},
			{"BRn x2f34", "\x09\x33", 1, lc3.RegisterPC, 0x3001, lc3.FlagZ},

			{"LEA R0, x3003", "\xE0\x02", 1, lc3.RegisterR0, 0x3003, lc3.FlagP},
			{"LEA R1, x2F35", "\xE3\x34", 1, lc3.RegisterR1, 0x2F35, lc3.FlagP},
			{"LEA R7, x3001", "\xEE\x00", 1, lc3.RegisterR7, 0x3001, lc3.FlagP},

			{"NOT R0, R0", "\x90\x3f", 1, lc3.RegisterR0, 0xffff, lc3.FlagN},
			{"LEA R3, x3039 + NOT R3, R1", "\xE6\x38\x96\xFF", 2, lc3.RegisterR3, 0xCFC6, lc3.FlagN},

			{"ADD R0, R0, R0", "\x10\x00", 1, lc3.RegisterR0, 0x0000, lc3.FlagZ},
			{"ADD R0, R0, #0", "\x10\x20", 1, lc3.RegisterR0, 0x0000, lc3.FlagZ},
			{"ADD R3, R2, #5", "\x16\x25", 1, lc3.RegisterR3, 0x0005, lc3.FlagP},
			{"ADD R5, R4, #-11", "\x1B\x35", 1, lc3.RegisterR5, 0xfff5, lc3.FlagN},
			{"ADD R7, R0, #-14 + ADD R3, R0, R7", "\x1E\x32\x16\x07", 2, lc3.RegisterR3, 0xfff2, lc3.FlagN},

			{"AND R0, R0, R0", "\x50\x00", 1, lc3.RegisterR0, 0x0000, lc3.FlagZ},
			{"AND R3, R7, #-22", "\x57\xEA", 1, lc3.RegisterR3, 0x0000, lc3.FlagZ},
			{"ADD R7, R0, #-14 + AND R3, R7, R7", "\x1E\x32\x57\xC7", 2, lc3.RegisterR3, 0xfff2, lc3.FlagN},

			{"JMP R3", "\xC0\x00", 1, lc3.RegisterPC, 0x0000, lc3.FlagZ},
			{"RET (JMP R7)", "\xC1\xC0", 1, lc3.RegisterPC, 0x0000, lc3.FlagZ},
			{"ADD R5 R4 #-15 + JMP R5", "\x1B\x31\xC1\x40", 2, lc3.RegisterPC, 0xfff1, lc3.FlagN},

			{"JSR x3001", "\x48\x00", 1, lc3.RegisterPC, 0x3001, lc3.FlagZ},
			{"JSRR R0; check R7", "\x40\x00", 1, lc3.RegisterR7, 0x3001, lc3.FlagZ},
			{"JSRR R0; check PC", "\x40\x00", 1, lc3.RegisterPC, 0x0000, lc3.FlagZ},
			{"ADD R3, R3, #14 + JSRR R3", "\x16\xEE\x40\xC0", 2, lc3.RegisterPC, 0x000E, lc3.FlagP},

			{"LD R0, x3001", "\x20\x00", 1, lc3.RegisterR0, 0x0000, lc3.FlagZ},
			{"LD R5, x3003", "\x2A\x02\x00\x00\x00\x00\x01\x23", 1, lc3.RegisterR5, 0x0123, lc3.FlagP},
			{"LD R0, x2F06", "\x21\x05", 1, lc3.RegisterR0, 0x1234, lc3.FlagP},

			{"LDI R6, x3001", "\xA6\x00\x00\x00", 1, lc3.RegisterR3, 0x0000, lc3.FlagZ},
			{"LDI R8, x3002", "\xA8\x01\x00\x15\x30\x01", 1, lc3.RegisterR4, 0x0015, lc3.FlagP},

			{"LDR R0, R0, #0", "\x60\x00", 1, lc3.RegisterR0, 0x0000, lc3.FlagZ},
			{"LEA R1, x3002 + LDR R4, R1, #1", "\xE2\x01\x68\x41\x00\x00\xFF\xFF", 2, lc3.RegisterR4, 0xFFFF, lc3.FlagN},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				program := strings.NewReader("\x30\x00" + test.instruction)

				vm, err := lc3.NewVM(program, nil, nil)
				assertError(t, err, nil)
				vm.SetMemory(canaryAddress, canaryValue)

				for i := 0; i < test.steps; i++ {
					err = vm.Step()
					assertError(t, err, nil)
				}

				assertRegister(t, vm, test.reg, test.value)
				assertRegister(t, vm, lc3.RegisterCOND, test.flag)
			})
		}
	})

	t.Run("test ST/STI/STR instructions", func(t *testing.T) {
		testCases := []struct {
			name        string
			instruction string
			memory      uint16
			value       uint16
			flag        uint16
		}{
			{"NOP + ST R2, x3006", "\x00\x00\x34\x05", 0x3006, 0x0, lc3.FlagZ},
			{"ADD R5 R4 #-14 + ST R5, x3006", "\x1B\x32\x3A\x40", 0x3042, 0xFFF2, lc3.FlagN},

			{"NOP + STI R0, x3002", "\x00\x00\xB0\x00", 0x0000, 0x0, lc3.FlagZ},
			{"ADD R7, R4, #-1 + STI R7, x3003", "\x1F\x3F\xBE\x01\x00\x00\x12\x34", 0x1234, 0xFFFF, lc3.FlagN},

			{"NOP + STR R0, R0, #0", "\x00\x00\x70\x00", 0x0000, 0x0, lc3.FlagZ},
			{"ADD R7, R4, #8 + STR R7, R0, #1", "\x1F\x28\x7E\x12", 0x0012, 0x0008, lc3.FlagP},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				program := strings.NewReader("\x30\x00" + test.instruction)

				vm, err := lc3.NewVM(program, nil, nil)
				assertError(t, err, nil)

				err = vm.Step()
				assertError(t, err, nil)
				err = vm.Step()
				assertError(t, err, nil)

				assertMemory(t, vm, test.memory, test.value)
				assertRegister(t, vm, lc3.RegisterCOND, test.flag)
			})
		}
	})

	t.Run("test PUTS trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xE0\x01\xf0\x22\x00\x41\x00\x00")

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, nil, output)
		assertError(t, err, nil)

		// LEA R0, x3002
		err = vm.Step()
		assertError(t, err, nil)

		// PUTS (R0 == x3002 == "H\0")
		err = vm.Step()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.RegisterPC, 0x3002)

		assertString(t, "A", output.String())
	})

	t.Run("test HALT trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x25\x00\x00")

		vm, err := lc3.NewVM(program, nil, nil)
		assertError(t, err, nil)

		err = vm.Step()
		assertError(t, err, nil)
		assertState(t, vm.State(), lc3.StateHalted)
		assertRegister(t, vm, lc3.RegisterPC, 0x3001)

		// Can't step further
		err = vm.Step()
		assertIsError(t, err)
		assertRegister(t, vm, lc3.RegisterPC, 0x3001)
	})

	t.Run("test GETC trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x20")
		want := 'O'
		input := strings.NewReader(string(want))

		vm, err := lc3.NewVM(program, input, nil)
		assertError(t, err, nil)
		// To make sure that top bytes are also cleared.
		vm.SetRegister(lc3.RegisterR0, 0x1234)

		err = vm.Step()
		assertError(t, err, nil)
		assertState(t, vm.State(), lc3.StateRunning)
		assertRegister(t, vm, lc3.RegisterPC, 0x3001)
		assertRegister(t, vm, lc3.RegisterR0, uint16(want))
	})

	t.Run("test OUT trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x21")
		want := byte(0x41)

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, nil, output)
		assertError(t, err, nil)
		vm.SetRegister(lc3.RegisterR0, uint16(want))

		err = vm.Step()
		assertError(t, err, nil)
		assertString(t, output.String(), string([]byte{0x41}))
	})

	t.Run("execute hello-world.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/hello-world.obj")
		defer closer()

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(f, nil, output)
		assertError(t, err, nil)

		var start uint16 = 0x3000
		values := []uint16{0xe002, 0xf022, 0xf025, 0x0048,
			0x0065, 0x006c, 0x006c, 0x006f,
			0x0020, 0x0057, 0x006f, 0x0072,
			0x006c, 0x0064, 0x0021, 0x0000}
		assertRegister(t, vm, lc3.RegisterPC, start)
		for i, value := range values {
			assertMemory(t, vm, start+uint16(i), value)
		}

		// LEA R0, HELLO_STR
		// PUTS
		// HALT
		err = vm.Run()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.RegisterPC, start+3)
		assertRegister(t, vm, lc3.RegisterR0, 0x3003)
		assertRegister(t, vm, lc3.RegisterCOND, lc3.FlagP)
		assertString(t, "Hello World!", output.String())
		assertState(t, vm.State(), lc3.StateHalted)
	})

	t.Run("execute loop.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/loop.obj")
		defer closer()

		vm, err := lc3.NewVM(f, nil, nil)
		assertError(t, err, nil)

		err = vm.Run()
		assertError(t, err, nil)
		assertRegister(t, vm, lc3.RegisterPC, 0x3005)
		assertRegister(t, vm, lc3.RegisterR0, 10)
		assertRegister(t, vm, lc3.RegisterR1, 0)
		assertRegister(t, vm, lc3.RegisterCOND, lc3.FlagZ)
		assertState(t, vm.State(), lc3.StateHalted)
	})

	t.Run("execute reverse-string.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/reverse-string.obj")
		defer closer()

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(f, nil, output)
		assertError(t, err, nil)

		err = vm.Run()
		assertError(t, err, nil)
		assertString(t, "4321DCBA", output.String())
		assertState(t, vm.State(), lc3.StateHalted)
	})

	t.Run("test KBSR/KBDR memory registers", func(t *testing.T) {
		program := strings.NewReader("\x30\x00")
		want := 'A'
		input := strings.NewReader(string(want))

		vm, err := lc3.NewVM(program, input, nil)
		assertError(t, err, nil)

		// On memory read, KBSR highest-bit is set, KBDR contains wanted character.
		assertMemory(t, vm, lc3.MemoryKBSR, 0x8000)
		assertMemory(t, vm, lc3.MemoryKBDR, uint16(want))
		// Nothing more to read.
		assertMemory(t, vm, lc3.MemoryKBSR, 0x0000)
	})
}

func assertInitVM(t *testing.T, vm *lc3.VM, pc uint16) {
	t.Helper()

	assertState(t, vm.State(), lc3.StateRunning)
	for i := 0; i < lc3.MemorySize; i++ {
		assertMemory(t, vm, uint16(i), 0)
	}

	for reg := lc3.RegisterR0; reg < lc3.RegisterCOUNT; reg++ {
		if reg == lc3.RegisterPC {
			assertRegister(t, vm, reg, pc)
		} else if reg == lc3.RegisterCOND {
			assertRegister(t, vm, reg, lc3.FlagZ)
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
		t.Fatalf("expected register %q value to be 0x%x, got 0x%x", reg, want, got)
	}
}

func assertMemory(t *testing.T, vm *lc3.VM, address, want uint16) {
	t.Helper()

	got, err := vm.GetMemory(address)
	assertError(t, err, nil)
	if got != want {
		t.Fatalf("expected memory at 0x%x to have 0x%x, got 0x%x", address, want, got)
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
