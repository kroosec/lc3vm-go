package lc3

import (
	"fmt"
)

func (v *VM) trapGetc() error {
	char, err := v.getChar()
	if err != nil {
		return fmt.Errorf("couldn't read input: %v", err)
	}
	v.SetRegister(RegisterR0, uint16(char))
	return nil
}

func (v *VM) getChar() (byte, error) {
	char := make([]byte, 1)
	n, err := v.input.Read(char)
	if n == 0 || err != nil {
		return 0, err
	}
	return char[0], nil
}

func (v *VM) trapOut() error {
	char := v.GetRegister(RegisterR0) & 0xff
	if _, err := v.output.Write([]byte{byte(char)}); err != nil {
		return fmt.Errorf("couldn't write output %c: %v", char, err)
	}
	return nil
}

func (v *VM) trapHalt() {
	v.state = StateHalted
}

func (v *VM) trapPuts() error {
	address := v.GetRegister(RegisterR0)

	var out []byte
	for {
		value, err := v.GetMemory(address)
		if err != nil {
			return err
		}
		if value == 0 {
			break
		}

		if value > 0xff {
			return fmt.Errorf("Invalid character in string: 0x%x", value)
		}

		out = append(out, byte(value))
		if address == UserMemoryLimit {
			break
		}
		address++
	}

	if _, err := v.output.Write(out); err != nil {
		return fmt.Errorf("Couldn't write output %v: %v", out, err)
	}
	return nil
}

func (v *VM) peekChar() bool {
	_, err := v.input.Peek(1)
	return err == nil
}