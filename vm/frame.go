package vm

import (
	"github.com/ioanzicu/monkeyd/code"
	"github.com/ioanzicu/monkeyd/object"
)

type Frame struct {
	fn          *object.CompiledFunction
	ip          int
	basePointer int // frame pointer - for reference while executing a function
}

func NewFrame(fn *object.CompiledFunction, basePointer int) *Frame {
	return &Frame{
		fn:          fn,
		ip:          -1,
		basePointer: basePointer,
	}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fn.Instructions
}
