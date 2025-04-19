package vm

import (
	"fmt"

	"github.com/ioanzicu/monkeyd/code"
	"github.com/ioanzicu/monkeyd/compiler"
	"github.com/ioanzicu/monkeyd/object"
)

// 		STACK
//
// 		3 > 1
//  ---------------
// | OpConstant 0  | <- Load 3
//  ---------------
// | OpConstant 1  | <- Load 1
//  ---------------
// | OpGreaterThan | <- Load >
//  ---------------

// 		20 + 11
//  ---------------
// | OpConstant 0  | <- Load 20
//  ---------------
// | OpConstant 1  | <- Load 11
//  ---------------
// | 	 OpAdd 	   | <- Load +
//  ---------------

// 		20 - 11
//  ---------------
// | OpConstant 0  | <- Load 20
//  ---------------
// | OpConstant 1  | <- Load 11
//  ---------------
// | 	 OpSub 	   | <- Load -
//  ---------------

//
//
// 			CONDITIONAL JUMPS

//  	 ----------------------------
// 0000 | 	      OpConstant 0  	 |
//  	 ----------------------------
// 0001 | 	      OpConstant 1  	 |
//  	 ----------------------------
// 0002	| 	   	 OpGreaterThan 	 	 |
//  	 ----------------------------
//
//    	 ----------------------------
// 0003 | 	JUMP_IF_NOT_TRUE 0008  	 |
//  	 ----------------------------
//
//  	 ----------------------------
// 0004 | 	      OpConstant 2  	 |
//  	 ----------------------------
// 0005 | 	      OpConstant 3  	 |
//  	 ----------------------------
// 0006	| 	   	  	OpAdd 	 	     |
//  	 ----------------------------
//
//    	 ----------------------------
// 0007 | JUMP_NO_MATTER_WHAT 0011   |
//  	 ----------------------------
//
//  	 ----------------------------
// 0008 | 	      OpConstant 4  	 |
//  	 ----------------------------
// 0009 | 	      OpConstant 5  	 |
//  	 ----------------------------
// 0010	| 	   	  	OpMinus 	 	 |
//  	 ----------------------------
//
//
// 			KEEPING TRACK OF NAMES
//
//
//  	 ----------------------------
// 	    | 	      OpConstant 0  	 | <- Load the "33" onto the stack
//  	 ----------------------------
//      | 	     OpSetGlobal 0  	 | <- Bind value on stack to 0
//  	 ----------------------------
//
//  	 ----------------------------
// 	    | 	      OpConstant 1  	 | <- Load the "66" onto the stack
//  	 ----------------------------
//      | 	     OpSetGlobal 1  	 | <- Bind value on stack to 1
//  	 ----------------------------
//
//  	 ----------------------------
// 	    | 	     OpGetGlobal 1  	 | <- Push the global bound to 1
//  	 ----------------------------
//      | 	     OpGetGlobal 0  	 | <- Push the global bound to 0
//  	 ----------------------------
//      | 	     	 OpAdd     	     | <- Add them together
//  	 ----------------------------
//      | 	     OpSetGlobal 2  	 | <- Bind value on stack to 2
//  	 ----------------------------

const (
	StackSize   = 2048
	GlobalsSize = 65536
)

// Immutable unique values
// We will compare only the pointers
// without unwrapping the value the objects are pointing at
var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
	Null  = &object.Null{}
)

type VM struct {
	constants    []object.Object
	instructions code.Instructions
	stack        []object.Object
	sp           int // Always points to the next value. Top of stack is stack[sp - 1]

	globals []object.Object
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globals: make([]object.Object, GlobalsSize),
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = s
	return vm
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) Run() error {

	for ip := 0; ip < len(vm.instructions); ip++ {
		// FETCH
		op := code.Opcode(vm.instructions[ip])

		// DECODE & EXECUTE
		switch op {

		case code.OpConstant:
			// 1. DECODE the operands in the bytecode, after the Opcode
			constIndex := code.ReadUint16(vm.instructions[ip+1:])
			// 2. Skip over two bytes of the operand in the next cycle
			ip += 2

			// EXECUTE
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.pop()

		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		case code.OpJump:
			// 1. Decode the operand right after the Opcode
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			// 2. Set instruction pointer to the target of jump
			ip = pos - 1

		case code.OpJumpNotTruthy:
			// 1. Decode the operand right after the Opcode
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			// 2. Skip over two bytes of the operand in the next cycle
			// since OpJumpNotTruthy has OperandWidths of 2 bytes
			ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				ip = pos - 1
			}

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2

			vm.globals[globalIndex] = vm.pop()

		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2

			err := vm.push(vm.globals[globalIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip += 2

			array := vm.buildArray(vm.sp-numElements, vm.sp)
			vm.sp = vm.sp - numElements

			err := vm.push(array)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)

	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}

	return &object.Array{Elements: elements}
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {

	case *object.Boolean:
		return obj.Value

	case *object.Null:
		return false

	default:
		return true
	}
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	// DECODE
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	switch {

	case leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ:
		return vm.executeBinaryIntegerOperation(op, left, right)

	case leftType == object.STRING_OBJ && rightType == object.STRING_OBJ:
		return vm.executeBinaryStringOperation(op, left, right)
	default:
		return fmt.Errorf("unsupported types for binray operation: %s %s", leftType, rightType)
	}
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64

	// EXECUTE
	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeBinaryStringOperation(op code.Opcode, left, right object.Object) error {
	if op != code.OpAdd {
		return fmt.Errorf("unknown string operator: %d", op)
	}

	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	return vm.push(&object.String{Value: leftValue + rightValue})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeIntegerComparison(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue == leftValue))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(rightValue != leftValue))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}
	return False
}

// executeBangOperator - pop and negate (flip) the result
func (vm *VM) executeBangOperator() error {
	operand := vm.pop()

	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	case Null:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported type for negation: %s", operand.Type())
	}

	value := operand.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -value})
}
