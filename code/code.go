package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			continue
		}

		operands, read := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))

		i += 1 + read
	}

	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n", len(operands), operandCount)
	}

	switch operandCount {

	case 0:
		return def.Name

	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])

	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])

	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}

// Opcode - one byte wide
// has a unique value
// first byte in the instruction
type Opcode byte

const (
	OpConstant Opcode = iota

	OpPop

	OpAdd
	OpSub
	OpMul
	OpDiv

	OpTrue
	OpFalse

	OpEqual
	OpNotEqual
	OpGreaterThan

	OpMinus
	OpBang

	OpJumpNotTruthy
	OpJump

	OpNull

	// Global scope variable name bindings
	OpGetGlobal
	OpSetGlobal

	// Composite Data Types
	OpArray
	OpHash

	OpIndex

	OpCall        // start executing the *object.CompiledFunction sitting on top of the stack
	OpReturnValue // return the value on top of the stack to the calling context and to resume execution there
	OpReturn      // no explicit value to return but an implicit vm.Null

	// Local scope variable name bindings
	OpGetLocal
	OpSetLocal

	// Builtin Funcitions Instructions
	OpGetBuiltin

	OpClosure
	OpGetFree
	OpCurrentClosure
)

type Definition struct {
	Name          string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition{
	OpConstant: &Definition{
		Name:          "OpConstant",
		OperandWidths: []int{2}, // 2^16 2 bytes - 0..65535
	},
	OpAdd: &Definition{
		Name:          "OpAdd",
		OperandWidths: []int{},
	},
	OpPop: &Definition{
		Name:          "OpPop",
		OperandWidths: []int{},
	},
	OpSub: &Definition{
		Name:          "OpSub",
		OperandWidths: []int{},
	},
	OpMul: &Definition{
		Name:          "OpMul",
		OperandWidths: []int{},
	},
	OpDiv: &Definition{
		Name:          "OpDiv",
		OperandWidths: []int{},
	},
	OpFalse: &Definition{
		Name:          "OpFalse",
		OperandWidths: []int{},
	},
	OpTrue: &Definition{
		Name:          "OpTrue",
		OperandWidths: []int{},
	},
	OpEqual: &Definition{
		Name:          "OpEqual",
		OperandWidths: []int{},
	},
	OpNotEqual: &Definition{
		Name:          "OpNotEqual",
		OperandWidths: []int{},
	},
	OpGreaterThan: &Definition{
		Name:          "OpGreaterThan",
		OperandWidths: []int{},
	},
	OpMinus: &Definition{
		Name:          "OpMinus",
		OperandWidths: []int{},
	},
	OpBang: &Definition{
		Name:          "OpBang",
		OperandWidths: []int{},
	},
	// not false or null
	OpJumpNotTruthy: &Definition{
		Name:          "OpJumpNotTruthy",
		OperandWidths: []int{2},
	},
	// jump to the instruction offset
	OpJump: &Definition{
		Name:          "OpJump",
		OperandWidths: []int{2},
	},
	OpNull: &Definition{
		Name:          "OpNull",
		OperandWidths: []int{},
	},
	OpGetGlobal: &Definition{
		Name:          "OpGetGlobal",
		OperandWidths: []int{2},
	},
	OpSetGlobal: &Definition{
		Name:          "OpSetGlobal",
		OperandWidths: []int{2},
	},
	OpArray: &Definition{
		Name:          "OpArray",
		OperandWidths: []int{2},
	},
	OpHash: &Definition{
		Name:          "OpHash",
		OperandWidths: []int{2},
	},
	OpIndex: &Definition{
		Name:          "OpIndex",
		OperandWidths: []int{},
	},
	OpCall: &Definition{
		Name:          "OpCall",
		OperandWidths: []int{1}, // len of function arguments - max 256
	},
	OpReturnValue: &Definition{
		Name:          "OpReturnValue",
		OperandWidths: []int{},
	},
	OpReturn: &Definition{
		Name:          "OpReturn",
		OperandWidths: []int{},
	},
	OpGetLocal: &Definition{
		Name:          "OpGetLocal",
		OperandWidths: []int{1}, // 2^8 = up to 256 local variable
	},
	OpSetLocal: &Definition{
		Name:          "OpSetLocal",
		OperandWidths: []int{1},
	},
	OpGetBuiltin: &Definition{
		Name:          "OpGetBuiltin",
		OperandWidths: []int{1}, // 2^8 = up to 256 builtin functions
	},
	OpClosure: &Definition{
		Name: "OpClosure",
		OperandWidths: []int{
			2, // bytes - constant index, where in the constant pool we can find
			// the *object.CompiledFunction that’s to be converted into a closure
			// 2 bytes wide because the OpConstant is 2 bytes wide
			1, // max 256 - free variables sit on the stack and need
			// to be transferred to the about-to-be-created closure
		},
	},
	OpGetFree: &Definition{
		Name:          "OpGetFree",
		OperandWidths: []int{1},
	},
	OpCurrentClosure: &Definition{
		Name:          "OpCurrentClosure",
		OperandWidths: []int{},
	},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}

	return def, nil
}

// Make - encode the operands of a bytecode instruction
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

		case 1:
			instruction[offset] = byte(o)

		}
		offset += width
	}

	return instruction
}

// ReadOperands - takes a Definition and Instructions and decode them
// Oposite to the Make function
func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {

		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))

		case 1:
			operands[i] = int(ReadUint8(ins[offset:]))

		}

		offset += width
	}

	return operands, offset
}

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins)
}

func ReadUint8(ins Instructions) uint8 {
	return uint8(ins[0])
}
