package code

import (
	"encoding/binary"
	"fmt"
)

type Instructions []byte

// Opcode - one byte wide
// has a unique value
// first byte in the instruction
type Opcode byte

const (
	OpConstant Opcode = iota
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant: &Definition{
		Name:          "OpConstant",
		OperandWidths: []int{2}, // 2 bytes - 0..65535
	},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}

	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1 // for OpCode
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op) // set OpCode

	offset := 1 // since OpCode is already there
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			// split the value 16 bits 2 bytes - into 2 values of 8 bits - 1 byte each
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))
		}
		offset += width
	}

	return instruction
}
