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
//
//
// 			COMPILING FUNCTIONS
//
// 			fn() { return 3 + 30 }
//
//  	 ----------------------------
// 	    | 	      OpConstant 0  	 | <- Load  3 on to the stack
//  	 ----------------------------
//      | 	      OpConstant 1  	 | <- Load 30 on to the stack
//  	 ----------------------------
//      | 	     	 OpAdd     	     | <- Add them together
//  	 ----------------------------
//      | 	     OpReturnValue  	 | <- Return value on top of stack
//  	 ----------------------------

const (
	StackSize   = 2048
	GlobalsSize = 65536
	MaxFrames   = 1024
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
	constants []object.Object

	stack []object.Object
	sp    int // Always points to the next value. Top of stack is stack[sp - 1]

	globals []object.Object

	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globals: make([]object.Object, GlobalsSize),

		frames:      frames,
		framesIndex: 1,
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
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		// FETCH
		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		// DECODE & EXECUTE
		switch op {

		case code.OpConstant:

			// 1. DECODE the operands in the bytecode, after the Opcode
			constIndex := code.ReadUint16(ins[ip+1:])

			// 2. Skip over two bytes of the operand in the next cycle
			vm.currentFrame().ip += 2

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
			pos := int(code.ReadUint16(ins[ip+1:]))

			// 2. Set instruction pointer to the target of jump
			vm.currentFrame().ip = pos - 1

		case code.OpJumpNotTruthy:

			// 1. Decode the operand right after the Opcode
			pos := int(code.ReadUint16(ins[ip+1:]))

			// 2. Skip over two bytes of the operand in the next cycle
			// since OpJumpNotTruthy has OperandWidths of 2 bytes
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			vm.globals[globalIndex] = vm.pop()

		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			err := vm.push(vm.globals[globalIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			array := vm.buildArray(vm.sp-numElements, vm.sp)
			vm.sp = vm.sp - numElements

			err := vm.push(array)
			if err != nil {
				return err
			}

		case code.OpHash:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-numElements, vm.sp)
			if err != nil {
				return err
			}
			vm.sp = vm.sp - numElements

			err = vm.push(hash)
			if err != nil {
				return err
			}

		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}

		case code.OpCall:
			fn, ok := vm.stack[vm.sp-1].(*object.CompiledFunction)
			if !ok {
				return fmt.Errorf("calling non-function")
			}
			frame := NewFrame(fn)
			vm.pushFrame(frame)

		case code.OpReturnValue:
			returnValue := vm.pop()

			vm.popFrame()
			vm.pop()

			err := vm.push(returnValue)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {

	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)

	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)

	default:
		return fmt.Errorf("index operatoer not supported: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	// check for out of bounds
	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(arrayObject.Elements[i])
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.push(Null)
	}

	return vm.push(pair.Value)
}

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)

	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}

	return &object.Array{Elements: elements}
}

func (vm *VM) buildHash(startIndex, endIndex int) (object.Object, error) {
	hashedPairs := make(map[object.HashKey]object.HashPair)

	for i := startIndex; i < endIndex; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]

		pair := object.HashPair{Key: key, Value: value}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable as hash key: %s", key.Type())
		}

		hashedPairs[hashKey.HashKey()] = pair
	}

	return &object.Hash{Pairs: hashedPairs}, nil
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

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}
