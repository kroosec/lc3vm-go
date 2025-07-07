package lc3_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
				assert.Error(t, err)
			})
		}
	})

	t.Run("load an empty program and check PC", func(t *testing.T) {
		var pc uint16 = 0x3000
		program := strings.NewReader("\x30\x00")

		vm, err := lc3.NewVM(program, nil, nil)
		assert.NoError(t, err)
		assertInitVM(t, vm, pc)
	})

	t.Run("load a simple program and check memory", func(t *testing.T) {
		var start uint16 = 0x3000
		program := strings.NewReader("\x30\x00\x12\x34")

		vm, err := lc3.NewVM(program, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, start, vm.GetRegister(lc3.RegisterPC))
		val, err := vm.GetMemory(start)
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x1234), val)
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
				assert.NoError(t, err)
				vm.SetMemory(canaryAddress, canaryValue)

				for i := 0; i < test.steps; i++ {
					err = vm.Step()
					assert.NoError(t, err)
				}

				assert.Equal(t, test.value, vm.GetRegister(test.reg))
				assert.Equal(t, test.flag, vm.GetRegister(lc3.RegisterCOND))
			})
		}
	})

	t.Run("test NOT instruction with invalid trailing bits", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\x90\x00") // NOT R0, R0 with invalid trailing bits

		vm, err := lc3.NewVM(program, nil, nil)
		assert.NoError(t, err)

		err = vm.Step()
		assert.Error(t, err)
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
				assert.NoError(t, err)

				err = vm.Step()
				assert.NoError(t, err)
				err = vm.Step()
				assert.NoError(t, err)

				val, err := vm.GetMemory(test.memory)
				assert.NoError(t, err)
				assert.Equal(t, test.value, val)
				assert.Equal(t, test.flag, vm.GetRegister(lc3.RegisterCOND))
			})
		}
	})

	t.Run("test PUTS trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xE0\x01\xf0\x22\x00\x41\x00\x00")

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, nil, output)
		assert.NoError(t, err)

		// LEA R0, x3002
		err = vm.Step()
		assert.NoError(t, err)

		// PUTS (R0 == x3002 == "H\0")
		err = vm.Step()
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x3002), vm.GetRegister(lc3.RegisterPC))

		assert.Equal(t, "A", output.String())
	})

	t.Run("test PUTS trap with invalid character", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xE0\x01\xf0\x22\x00\x41\x01\x00") // LEA R0, x3002; PUTS; 'A', 0x0100 (invalid)

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, nil, output)
		assert.NoError(t, err)

		// LEA R0, x3002
		err = vm.Step()
		assert.NoError(t, err)

		// PUTS (R0 == x3002 == "A", 0x0100)
		err = vm.Step()
		assert.Error(t, err) // Expect an error due to invalid character
	})

	t.Run("test HALT trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x25\x00\x00")

		vm, err := lc3.NewVM(program, nil, nil)
		assert.NoError(t, err)

		err = vm.Step()
		assert.NoError(t, err)
		assert.Equal(t, lc3.StateHalted, vm.State())
		assert.Equal(t, uint16(0x3001), vm.GetRegister(lc3.RegisterPC))

		// Can't step further
		err = vm.Step()
		assert.Error(t, err)
		assert.Equal(t, uint16(0x3001), vm.GetRegister(lc3.RegisterPC))
	})

	t.Run("test GETC trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x20")
		want := 'O'
		input := strings.NewReader(string(want))

		vm, err := lc3.NewVM(program, input, nil)
		assert.NoError(t, err)
		// To make sure that top bytes are also cleared.
		vm.SetRegister(lc3.RegisterR0, 0x1234)

		err = vm.Step()
		assert.NoError(t, err)
		assert.Equal(t, lc3.StateRunning, vm.State())
		assert.Equal(t, uint16(0x3001), vm.GetRegister(lc3.RegisterPC))
		assert.Equal(t, uint16(want), vm.GetRegister(lc3.RegisterR0))
	})

	t.Run("test OUT trap", func(t *testing.T) {
		program := strings.NewReader("\x30\x00\xf0\x21")
		want := byte(0x41)

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(program, nil, output)
		assert.NoError(t, err)
		vm.SetRegister(lc3.RegisterR0, uint16(want))

		err = vm.Step()
		assert.NoError(t, err)
		assert.Equal(t, string([]byte{0x41}), output.String())
	})

	t.Run("execute hello-world.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/hello-world.obj")
		defer closer()

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(f, nil, output)
		assert.NoError(t, err)

		var start uint16 = 0x3000
		values := []uint16{0xe002, 0xf022, 0xf025, 0x0048,
			0x0065, 0x006c, 0x006c, 0x006f,
			0x0020, 0x0057, 0x006f, 0x0072,
			0x006c, 0x0064, 0x0021, 0x0000}
		assert.Equal(t, start, vm.GetRegister(lc3.RegisterPC))
		for i, value := range values {
			val, err := vm.GetMemory(start + uint16(i))
			assert.NoError(t, err)
			assert.Equal(t, value, val)
		}

		// LEA R0, HELLO_STR
		// PUTS
		// HALT
		err = vm.Run()
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x3003), vm.GetRegister(lc3.RegisterPC))
		assert.Equal(t, uint16(0x3003), vm.GetRegister(lc3.RegisterR0))
		assert.Equal(t, lc3.FlagP, vm.GetRegister(lc3.RegisterCOND))
		assert.Equal(t, "Hello World!", output.String())
		assert.Equal(t, lc3.StateHalted, vm.State())
	})

	t.Run("execute loop.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/loop.obj")
		defer closer()

		vm, err := lc3.NewVM(f, nil, nil)
		assert.NoError(t, err)

		err = vm.Run()
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x3005), vm.GetRegister(lc3.RegisterPC))
		assert.Equal(t, uint16(10), vm.GetRegister(lc3.RegisterR0))
		assert.Equal(t, uint16(0), vm.GetRegister(lc3.RegisterR1))
		assert.Equal(t, lc3.FlagZ, vm.GetRegister(lc3.RegisterCOND))
		assert.Equal(t, lc3.StateHalted, vm.State())
	})

	t.Run("execute reverse-string.obj program", func(t *testing.T) {
		f, closer := openTestfile(t, "testdata/reverse-string.obj")
		defer closer()

		output := bytes.NewBuffer([]byte{})
		vm, err := lc3.NewVM(f, nil, output)
		assert.NoError(t, err)

		err = vm.Run()
		assert.NoError(t, err)
		assert.Equal(t, "4321DCBA", output.String())
		assert.Equal(t, lc3.StateHalted, vm.State())
	})

	t.Run("test KBSR/KBDR memory registers", func(t *testing.T) {
		program := strings.NewReader("\x30\x00")
		want := 'A'
		input := strings.NewReader(string(want))

		vm, err := lc3.NewVM(program, input, nil)
		assert.NoError(t, err)

		// On memory read, KBSR highest-bit is set, KBDR contains wanted character.
		val, err := vm.GetMemory(lc3.MemoryKBSR)
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x8000), val)
		val, err = vm.GetMemory(lc3.MemoryKBDR)
		assert.NoError(t, err)
		assert.Equal(t, uint16(want), val)
		// Nothing more to read.
		val, err = vm.GetMemory(lc3.MemoryKBSR)
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x0000), val)
	})
}

func assertInitVM(t *testing.T, vm *lc3.VM, pc uint16) {
	t.Helper()

	assert.Equal(t, lc3.StateRunning, vm.State())
	for i := 0; i < lc3.MemorySize; i++ {
		val, err := vm.GetMemory(uint16(i))
		assert.NoError(t, err)
		assert.Equal(t, uint16(0), val)
	}

	for reg := lc3.RegisterR0; reg < lc3.RegisterCOUNT; reg++ {
		if reg == lc3.RegisterPC {
			assert.Equal(t, pc, vm.GetRegister(reg))
		} else if reg == lc3.RegisterCOND {
			assert.Equal(t, lc3.FlagZ, vm.GetRegister(reg))
		} else {
			assert.Equal(t, uint16(0), vm.GetRegister(reg))
		}
	}
}

func openTestfile(t *testing.T, name string) (*os.File, func() error) {
	t.Helper()

	f, err := os.Open(name)
	assert.NoError(t, err)

	return f, f.Close
}